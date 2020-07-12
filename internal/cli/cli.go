package cli

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/progrhyme/binq-gh/binqgh"
	"github.com/progrhyme/binq-gh/internal/erron"
	"github.com/progrhyme/binq/schema/item"
	"github.com/progrhyme/go-lv"
	"github.com/spf13/pflag"
)

const (
	exitOK = iota
	exitNG
)

const (
	envGitHubTokenKey          = "GITHUB_TOKEN"
	defaultLogLevel   lv.Level = lv.LInfo
)

var dlTimeout = 5 * time.Minute

type CLI struct {
	OutStream, ErrStream io.Writer
	InStream             io.Reader
	opts                 *option
}

type option struct {
	token, logLv       *string
	yes, help, version *bool
}

func NewCLI(outs, errs io.Writer, ins io.Reader) *CLI {
	return &CLI{OutStream: outs, ErrStream: errs, InStream: ins}
}

// update-manifest
func (c *CLI) Run(args []string) (exit int) {
	prog := filepath.Base(args[0])
	flags := pflag.NewFlagSet(prog, pflag.ExitOnError)
	flags.SetOutput(c.ErrStream)
	flags.Usage = func() { c.usage(flags, prog) }
	c.opts = mkopts(flags)
	flags.Parse(args)

	if *c.opts.help {
		flags.Usage()
		return exitOK
	}
	if *c.opts.version {
		fmt.Fprintf(c.OutStream, "Version: %s\n", binqgh.Version)
		return exitOK
	}
	c.configureLogging()

	if len(args) <= 1 {
		fmt.Fprintln(c.ErrStream, "Error! Target is not specified")
		flags.Usage()
		return exitNG
	}

	file := args[1]
	_, obj, err := readAndDecodeItemJSONFile(file)
	if err != nil {
		fmt.Fprintf(c.ErrStream, "Error! %v\n", err)
		return exitNG
	}

	param := item.ItemURLParam{
		OS: runtime.GOOS, Arch: runtime.GOARCH,
	}
	uri, err := obj.GetLatestURL(param)
	if err != nil {
		fmt.Fprintf(c.ErrStream, "Error! %v\n", err)
		return exitNG
	}
	fmt.Fprintf(c.ErrStream, "URL of Item: %s\n", uri)

	var owner, repo string
	re := regexp.MustCompile(`^https://github\.com/([\w\-]+)/([\w\-]+)/`)
	if re.MatchString(uri) {
		matched := re.FindStringSubmatch(uri)
		owner = matched[1]
		repo = matched[2]
		lv.Debugf("owner: %s, repo: %s", owner, repo)
	} else {
		fmt.Fprintf(c.ErrStream, "Error! URL doesn't look like one of GitHub: %s\n", uri)
		return exitNG
	}

	ctx := context.Background()
	ghClient := newGitHubClient(ctx, *c.opts.token)
	latestRelease, _, err := ghClient.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		fmt.Fprintf(c.ErrStream, "Error! %v\n", err)
		return exitNG
	}

	ghLatestVersion := releaseNameToVersion(*latestRelease.Name)
	if obj.Latest.Version == ghLatestVersion {
		fmt.Fprintf(c.OutStream, "Manifest is up-to-date. Nothing to do\n")
		return exitOK
	}

	fmt.Fprintf(c.ErrStream, "New Version is found: %s\n", ghLatestVersion)

	var sums []item.ItemChecksum
	if len(latestRelease.Assets) > 0 {
		fmt.Fprintf(c.ErrStream, "Fetching bundled assets ...\n")
		var checksums *github.ReleaseAsset
		var checksumOfNameAndType = nestedMapOfReleaseAsset{}
		var downloads []*github.ReleaseAsset
		for _, asset := range latestRelease.Assets {
			lv.Debugf("Name: %s, DL URL: %s", *asset.Name, *asset.BrowserDownloadURL)
			if *asset.Name == "checksums.txt" {
				checksums = asset
			} else if strings.HasSuffix(*asset.Name, ".sha256") {
				name := strings.TrimSuffix(*asset.Name, ".sha256")
				checksumOfNameAndType.set(name, "sha256", asset)
			} else if strings.HasSuffix(*asset.Name, ".md5") {
				name := strings.TrimSuffix(*asset.Name, ".md5")
				checksumOfNameAndType.set(name, "md5", asset)
			} else {
				downloads = append(downloads, asset)
			}
		}

		if checksums != nil {
			fmt.Fprintf(c.ErrStream, "GET %s\n", *checksums.BrowserDownloadURL)
			sums, err = c.getChecksumsByAssetURL(*checksums.BrowserDownloadURL)
			if err != nil {
				return exitNG
			}
		}
		if len(sums) == 0 && len(checksumOfNameAndType) > 0 {
			sums, err = c.getChecksumsByNestedMap(checksumOfNameAndType)
			if err != nil {
				return exitNG
			}
		}
		if len(sums) == 0 && len(downloads) > 0 {
			for _, dl := range downloads {
				sum, err := c.getChecksumOfAsset(dl)
				if err != nil {
					return exitNG
				}
				sums = append(sums, sum)
			}
		}
	} else {
		lv.Warnf("Release has no asset to be downloaded")
	}

	err = c.runBinqReviseCommand(file, ghLatestVersion, sums)
	if err != nil {
		fmt.Fprintf(c.ErrStream, "Error! binq command failed\n")
		return exitNG
	}

	return exitOK
}

func (c *CLI) usage(fs *pflag.FlagSet, prog string) {
	fmt.Fprintf(c.ErrStream, "Usage of %s:\n", prog)
	fs.PrintDefaults()
}

func mkopts(fs *pflag.FlagSet) *option {
	return &option{
		token:   fs.StringP("token", "t", "", "# GitHub API Token"),
		yes:     fs.BoolP("yes", "y", false, "# Update JSON file without confirmation"),
		version: fs.BoolP("version", "v", false, "# Show version"),
		help:    fs.BoolP("help", "h", false, "# Show help"),
		logLv:   fs.StringP("log-level", "L", "", "# Log level (debug,info,notice,warn,error)"),
	}
}

func (c *CLI) configureLogging() {
	lv.Configure(c.ErrStream, defaultLogLevel, 0)
	var level lv.Level
	if *c.opts.logLv != "" {
		level = lv.WordToLevel(*c.opts.logLv)
		if level == 0 {
			lv.Warnf("Unknown log level: %s", *c.opts.logLv)
		} else {
			lv.SetLevel(level)
		}
	}
}

func readAndDecodeItemJSONFile(file string) (raw []byte, obj *item.Item, err error) {
	raw, _err := ioutil.ReadFile(file)
	if _err != nil {
		err = erron.Errorwf(_err, "Can't read item file: %s", file)
		return raw, obj, err
	}
	obj, _err = item.DecodeItemJSON(raw)
	if _err != nil {
		err = erron.Errorwf(_err, "Failed to decode Item JSON: %s", file)
		return raw, obj, err
	}
	return raw, obj, nil
}

func releaseNameToVersion(relver string) (version string) {
	re := regexp.MustCompile(`\d(?:[\d\.]*\d)?(?:\-[\w\-]+)?`)
	if re.MatchString(relver) {
		matched := re.FindStringSubmatch(relver)
		return matched[0]
	}

	lv.Errorf("Can't parse release version as version: %s", relver)
	return ""
}

func (c *CLI) getChecksumsByAssetURL(uri string) (sums []item.ItemChecksum, err error) {
	res, err := doHTTPGetRequest(uri, map[string]string{}, 5*time.Second)
	if err != nil {
		return sums, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Fprintf(c.ErrStream, "Error! HTTP response is not OK. Code: %d\n", res.StatusCode)
		return sums, err
	}

	for {
		var sum, file string
		_, err := fmt.Fscanln(res.Body, &sum, &file)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(c.ErrStream, "Error! Failed to read response body. %v", err)
			return sums, err
		}
		if len(sum) == 64 {
			lv.Infof("file: %s, sha256: %s", file, sum)
			sums = append(sums, item.ItemChecksum{File: file, SHA256: sum})
		} else {
			lv.Warnf("Unknown checksum format. File: %s, Value: %s", file, sum)
		}
	}

	return sums, nil
}

func (c *CLI) getChecksumsByNestedMap(nm nestedMapOfReleaseAsset) (
	sums []item.ItemChecksum, err error) {

	for name, stash := range nm {
		lv.Debugf("getChecksumsByNestedMap: name: %s", name)
		var asset *github.ReleaseAsset
		var kind item.ChecksumType
		// sha256 takes first place
		if stash["sha256"] != nil {
			asset = stash["sha256"]
			kind = item.ChecksumTypeSHA256
		} else if stash["md5"] != nil {
			asset = stash["md5"]
			kind = item.ChecksumTypeMD5
		} else {
			// Unexpected
			err = fmt.Errorf("Unknown type of stash: %+v, name: %s", stash, name)
			lv.Errorf("%s", err)
			return sums, err
		}
		val, err := c.getChecksumValueByAssetURL(*asset.BrowserDownloadURL)
		if err != nil {
			lv.Errorf("Failed to fetch checksum value. File: %s, URL: %s, Error: %v",
				name, *asset.BrowserDownloadURL, err)
			return sums, err
		}
		sum := item.ItemChecksum{File: name}
		sum.SetSum(val, kind)
		sums = append(sums, sum)
	}

	return sums, nil
}

// getChecksumValueByAssetURL returns first word of response text
func (c *CLI) getChecksumValueByAssetURL(uri string) (val string, err error) {
	res, err := doHTTPGetRequest(uri, map[string]string{}, 5*time.Second)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Fprintf(c.ErrStream, "Error! HTTP response is not OK. Code: %d\n", res.StatusCode)
		return "", err
	}

	scanner := bufio.NewScanner(res.Body)
	scanner.Split(bufio.ScanWords)
	scanner.Scan()
	if err = scanner.Err(); err != nil {
		fmt.Fprintf(c.ErrStream, "Error! Failed to read response body. %v", err)
		return val, err
	}
	val = scanner.Text()
	return val, nil
}

func (c *CLI) getChecksumOfAsset(asset *github.ReleaseAsset) (sum item.ItemChecksum, err error) {
	fmt.Fprintf(c.ErrStream, "GET %s\n", *asset.BrowserDownloadURL)
	res, err := doHTTPGetRequest(*asset.BrowserDownloadURL, map[string]string{}, dlTimeout)
	if err != nil {
		return sum, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Fprintf(c.ErrStream, "Error! HTTP response is not OK. Code: %d\n", res.StatusCode)
		return sum, err
	}

	tmpdir, err := ioutil.TempDir(os.TempDir(), "binq-gh.*")
	if err != nil {
		fmt.Fprintf(c.ErrStream, "Error! Failed to create tempdir. %v\n", err)
		return sum, err
	}
	defer os.RemoveAll(tmpdir)
	dlPath := filepath.Join(tmpdir, *asset.Name)
	dlFile, err := os.Create(dlPath)
	if err != nil {
		fmt.Fprintf(c.ErrStream, "Failed to open file: %s", dlPath)
		return sum, err
	}
	defer dlFile.Close()

	hasher := sha256.New()
	tee := io.TeeReader(res.Body, hasher)
	_, err = io.Copy(dlFile, tee)
	if err != nil {
		fmt.Fprintf(c.ErrStream, "Error! Failed to read HTTP response. %v\n", err)
		return sum, err
	}
	lv.Infof("Saved file %s", dlPath)

	cksum := hex.EncodeToString(hasher.Sum(nil))
	lv.Debugf("Sum: %s", cksum)

	return item.ItemChecksum{File: *asset.Name, SHA256: cksum}, nil
}

func (c *CLI) runBinqReviseCommand(file, version string, sums []item.ItemChecksum) (err error) {
	args := []string{"revise", file, "--version", version}
	if len(sums) > 0 {
		var argSums []string
		for _, sum := range sums {
			argSums = append(argSums, fmt.Sprintf("%s:%s", sum.File, sum.SHA256))
		}
		args = append(args, []string{"--sum", strings.Join(argSums, ",")}...)
	}
	if *c.opts.yes {
		args = append(args, "--yes")
	}
	if *c.opts.logLv != "" {
		args = append(args, []string{"--log-level", *c.opts.logLv}...)
	}
	binq := "binq"
	if os.Getenv(binqgh.EnvBinqPath) != "" {
		binq = os.Getenv(binqgh.EnvBinqPath)
	}
	cmd := exec.Command(binq, args...)
	cmd.Stdin = c.InStream
	cmd.Stdout = c.OutStream
	cmd.Stderr = c.ErrStream
	fmt.Fprintf(c.ErrStream, "[RUN] %s\n", cmd)

	return cmd.Run()
}
