package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	v := version()

	switch cmd {
	case "ns":
		fmt.Println(namespace(v))
		break
	case "version":
		fmt.Println(printVersion(v))
		break
	default:
		fmt.Println("Use: " + os.Args[0] + " version")
		fmt.Println("Use: " + os.Args[0] + " ns")
	}
}

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	if nil != err {
		return "", err
	}

	return out.String(), nil
}

func version() Version {
	var version Version = Version{
		versionType: INIT,
	}

	commitSha, err := git("log", "-1", "--format=%H")
	if nil != err {
		return version
	} else {
		version.commitSha = commitSha
	}

	commitDate, err := git("show", "-s", "--format=%ct")
	if nil != err {
		return version
	} else {
		commitDate = strings.Trim(commitDate, "\n")
		timestamp, err := strconv.ParseInt(commitDate, 10, 64)
		if err != nil {
			panic(err)
		}

		version.date = time.Unix(timestamp, 0)
	}

	versionTag, err := git("describe", "--match v*", "--exact", "--tags")

	if nil == err {
		v, err := createVersion(versionTag, ".")
		if nil == err {
			version.versionType = RELEASE
			version.patch = v.patch
			version.minor = v.minor
			version.major = v.major
			return version
		}
	}

	rcBranch, err := getReleaseBranch()

	if nil != err {
		version.versionType = FEATURE

		return version
	} else {
		version.minor = rcBranch.minor
		version.major = rcBranch.major
		version.versionType = RC
	}

	prevRelease, err := getPrevRelease()

	if nil == err {
		version.patch = prevRelease.patch + 1
	} else {
		version.patch = 1
	}

	return version
}

func getReleaseBranch() (Version, error) {
	var version Version
	rcBranch, err := git("describe", "--match", "*-stable", "--all", "--abbrev=0", "--first-parent")
	if nil != err {
		return version, err
	}
	rcBranchRegexp := regexp.MustCompile("(?m)(0|[1-9]\\d*)-(0|[1-9]\\d*)-stable$")
	rcBranch = rcBranchRegexp.FindString(rcBranch)

	if 0 == len(rcBranch) {

		return version, errors.New("invalid rc branch name")
	}

	version, err = createVersion(rcBranch, "-")

	if nil != err {
		return version, err
	}

	return version, nil
}

func namespace(version Version) string {
	switch version.versionType {
	case RELEASE:
		return fmt.Sprintf("/v%d.%d.%d", version.major, version.minor, version.patch)
	case INIT:
		return "/v0.0.0"
	case RC:
		return fmt.Sprintf("/v%d.%d.%d:%s", version.major, version.minor, version.patch, version.commitSha)
	default:
		return fmt.Sprintf("/dev-%s:%s", version.date.Format("2006-01"), version.commitSha)
	}
}

func getPrevRelease() (Version, error) {
	tag, err := git("describe", "--match", "v*", "--tags", "--abbrev=0")
	var v Version

	if nil != err {
		return v, err
	}

	r := regexp.MustCompile("v(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)")
	version := r.FindString(tag)

	if 0 == len(version) {
		return v, errors.New("unsupported version tag")
	}

	v, err = createVersion(version, ".")

	if nil != err {
		return v, err
	}

	v.versionType = RELEASE

	return v, nil
}

type Version struct {
	versionType         VersionType
	major, minor, patch int
	commitSha           string
	date                time.Time
}

func createVersion(version string, sep string) (Version, error) {
	var v Version

	filter := regexp.MustCompile("(0|[1-9]\\d*)(\\" + sep + "(0|[1-9]\\d*)){0,2}")
	filteredString := filter.FindString(version)
	parts := strings.Split(filteredString, sep)

	switch len(parts) {
	case 3:
		patch, err := parseNumber(parts[2])
		if nil != err {
			return v, err
		}
		v.patch = patch
	case 2:
		minor, err := parseNumber(parts[1])
		if nil != err {
			return v, err
		}

		v.minor = minor
	case 1:
		major, err := parseNumber(parts[0])
		if nil != err {
			return v, err
		}
		v.major = major
		break
	default:
		return v, errors.New("cannot parse version")
	}

	return v, nil
}

func parseNumber(number string) (int, error) {
	n, err := strconv.ParseInt(number, 10, 32)

	if nil != err {
		return -1, err
	}

	return int(n), nil
}

func printVersion(version Version) string {
	switch version.versionType {
	case RELEASE:
		return fmt.Sprintf("v%d.%d.%d", version.major, version.minor, version.patch)
	case INIT:
		return "v0.0.0"
	case RC:
		return fmt.Sprintf("v%d.%d.%d-%s", version.major, version.minor, version.patch, version.commitSha[0:8])
	default:
		return fmt.Sprintf("dev-%s", version.commitSha[0:8])
	}
}

type VersionType string

const (
	INIT    = "init"
	RC      = "rc"
	RELEASE = "release"
	FEATURE = "feature"
)
