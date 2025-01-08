package errstack

import (
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"path/filepath"
	"testing"

	_ "git.elewise.com/elma365-libraries/json"
	_ "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	// Load the dirs under ./testdata
	p, err := filepath.Abs("./testdata")
	require.NoError(t, err)

	files, err := os.ReadDir(p)
	require.NoError(t, err)

	for _, f := range files {
		t.Run(f.Name(), func(t *testing.T) {
			if !f.IsDir() {
				t.Fatalf("cannot run on non-directory: %s", f.Name())
			}

			dirPath, err := filepath.Abs(path.Join("./testdata", f.Name()))
			require.NoError(t, err)

			configPath := path.Join(dirPath, ".errstack.yaml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				// There is no config
				analysistest.Run(t, dirPath, NewAnalyzer(NewDefaultConfig()))
			} else if err == nil {
				// A config file exists, use it
				configFile, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var config Config
				require.NoError(t, yaml.Unmarshal(configFile, &config))
				analysistest.Run(t, dirPath, NewAnalyzer(config))
			} else {
				require.FailNow(t, err.Error())
			}
		})
	}
}

func TestExternalPkg(t *testing.T) {
	dirPath, err := filepath.Abs(path.Join("./testdata", "external_pkg"))
	require.NoError(t, err)
	settings := NewDefaultConfig()
	settings.MaxStackDepth = 5
	analysistest.Run(t, dirPath, NewAnalyzer(settings))
}
