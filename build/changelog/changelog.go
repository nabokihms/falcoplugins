/*
Copyright (C) 2022 The Falco Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/pflag"
)

func git(args ...string) (output []string, err error) {
	fmt.Fprintln(os.Stderr, "git ", strings.Join(args, " "))
	stdout, err := exec.Command("git", args...).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, errors.New("git (" + exitErr.String() + "): " + string(exitErr.Stderr))
		}
		return nil, err
	}
	return strings.Split(string(stdout), "\n"), nil
}

// an empty prefix matches the last tag with no match filtering
func gitGetLatestTagWithPrefix(prefix string) (string, error) {
	args := []string{"describe", "--tags", "--abbrev=0"}
	if len(prefix) > 0 {
		args = append(args, "--match", prefix+"*")
	}
	tags, err := git(args...)
	if err != nil {
		return "", err
	}
	if len(tags) == 0 {
		return "", errors.New("git tag not found")
	}
	return tags[0], nil
}

// an empty tag lists commit from whole history
func gitListCommits(from, to string) ([]string, error) {
	revRange := ""
	if len(to) > 0 {
		revRange = to
	}
	if len(from) > 0 {
		if len(revRange) == 0 {
			revRange = "HEAD"
		}
		revRange = from + ".." + revRange
	}
	logs, err := git("log", revRange, "--oneline")
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func fail(err error) {
	fmt.Printf("error: %s\n", err)
	os.Exit(1)
}

func main() {
	var plugin string
	var from string
	var to string
	pflag.StringVar(&plugin, "plugin", "", "Name of the plugin to generate the changelog for")
	pflag.StringVar(&from, "from", "", "Tag/branch/hash from which start listing commits")
	pflag.StringVar(&to, "to", "HEAD", "Tag/branch/hash to which stop listing commits")
	pflag.Parse()

	// if from is not specified, we use the latest tag matching the plugin name
	if len(from) == 0 {
		prefix := ""
		if len(plugin) > 0 {
			prefix = plugin + "-"
		}
		tag, err := gitGetLatestTagWithPrefix(prefix)
		if err != nil {
			fmt.Fprintln(os.Stderr, "not tag with prefix '"+prefix+"' not found, using commits from whole history:", err.Error())
		} else {
			from = tag
		}
	}

	// get all commits
	commits, err := gitListCommits(from, to)
	if err != nil {
		fail(err)
	}

	var rgx *regexp.Regexp
	if len(plugin) > 0 {
		// craft a regex to filter all plugin-related commits that follow
		// the conventional commit format
		rgx, _ = regexp.Compile("^[a-f0-9]{7} [a-zA-Z]+\\(([a-zA-Z\\/]+\\/)?" + plugin + "(\\/[a-zA-Z\\/]+)?\\):.*")
	}

	for _, c := range commits {
		if len(c) > 0 && (rgx == nil || rgx.MatchString(c)) {
			fmt.Println("*", c)
		}
	}
}