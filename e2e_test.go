// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main_test

import (
	"bytes"
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var update = flag.Bool("update", false, "update golden files")

func TestCsvCommandE2E(t *testing.T) {
	workdirs := []string{
		"testdata/modules/hello01",
		"testdata/modules/cli02",
		"testdata/modules/vendored03",
		"testdata/modules/replace04",
	}
	originalWorkDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// This builds go-licenses CLI to temporary dir.
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	goLicensesPath := filepath.Join(tempDir, "go-licenses")
	cmd := exec.Command("go", "build", "-o", goLicensesPath)
	_, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Built go-licenses binary in %s.", goLicensesPath)
	for _, workdir := range workdirs {
		t.Run(workdir, func(t *testing.T) {
			err := os.Chdir(filepath.Join(originalWorkDir, workdir))
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "mod", "download")
			log, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("downloading go modules:\n%s", string(log))
			}
			cmd = exec.Command(goLicensesPath, "csv", ".")
			// Capture stderr to buffer.
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			t.Logf("%s $ go-licenses csv .", workdir)
			output, err := cmd.Output()
			if err != nil {
				t.Logf("\n=== start of log ===\n%s=== end of log ===\n\n\n", stderr.String())
				t.Fatalf("running go-licenses csv: %s. Full log shown above.", err)
			}
			got := string(output)
			goldenFilePath := "licenses.csv"
			if *update {
				err := ioutil.WriteFile(goldenFilePath, output, 0o600)
				if err != nil {
					t.Fatalf("writing golden file: %s", err)
				}
			}
			goldenBytes, err := ioutil.ReadFile(goldenFilePath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					t.Fatalf("reading golden file: %s. Create a golden file by running `go test --update .`", err)
				}
				t.Fatalf("reading golden file: %s", err)
			}
			golden := string(goldenBytes)
			if got != golden {
				t.Logf("\n=== start of log ===\n%s=== end of log ===\n\n\n", stderr.String())
				t.Fatalf("result of go-licenses csv does not match the golden file.\n"+
					"Diff -golden +got:\n%s\n"+
					"Update the golden by running `go test --update .`",
					cmp.Diff(golden, got))
			}
		})
	}
}
