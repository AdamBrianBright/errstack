package errstack

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
	_ "golang.org/x/tools/go/analysis/passes/ctrlflow"
	"gopkg.in/yaml.v3"
)

func TestAnalyzer(t *testing.T) {
	// Load the dirs under ./testdata
	testdata := analysistest.TestData()
	t.Chdir(testdata + "/src")

	files, err := os.ReadDir(testdata + "/src")
	require.NoError(t, err)

	for _, f := range files {
		if !f.IsDir() || f.Name() == "vendor" {
			continue
		}

		t.Run(f.Name(), func(t *testing.T) {
			dirPath, err := filepath.Abs(path.Join("./testdata", f.Name()))
			require.NoError(t, err)

			configPath := path.Join(dirPath, ".errstack.yaml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				// There is no config
				analysistest.Run(t, testdata, NewAnalyzer(NewDefaultConfig()), f.Name())
			} else if err == nil {
				// A config file exists, use it
				configFile, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var config Config
				require.NoError(t, yaml.Unmarshal(configFile, &config))
				analysistest.Run(t, testdata, NewAnalyzer(config), f.Name())
			} else {
				require.FailNow(t, err.Error())
			}
		})
	}
}

func TestExternalPkg(t *testing.T) {
	testdata := analysistest.TestData()
	t.Chdir(testdata + "/src")

	settings := NewDefaultConfig()
	t.Run("double_return_wrap", func(t *testing.T) {
		analysistest.Run(t, testdata, NewAnalyzer(settings), "double_return_wrap")
	})
}
