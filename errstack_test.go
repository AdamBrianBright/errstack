package errstack_test

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdamBrianBright/errstack/internal/config"
	"github.com/AdamBrianBright/errstack/internal/helpers"
	"github.com/AdamBrianBright/errstack/internal/passes/errstack"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	_ "golang.org/x/tools/go/analysis/passes/ctrlflow"
)

func chdir(t *testing.T, dir string) {
	t.Helper()
	currentDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.Chdir(currentDir)
		require.NoError(t, err)
	})
}

func TestAnalyzer(t *testing.T) {
	// Load the dirs under ./testdata
	testdata := analysistest.TestData()
	chdir(t, testdata+"/src")

	files, err := os.ReadDir(testdata + "/src")
	require.NoError(t, err)

	_ = config.Analyzer.Flags.Set(config.Debug, "true")
	t.Cleanup(func() {
		_ = config.Analyzer.Flags.Set(config.Debug, "")
	})

	for _, f := range files {
		if !f.IsDir() || f.Name() == "vendor" {
			continue
		}

		t.Run(f.Name(), func(t *testing.T) {
			dirPath, err := filepath.Abs(path.Join(testdata, "./src", f.Name()))
			require.NoError(t, err)

			configPath := path.Join(dirPath, ".errstack.yaml")
			_, err = os.Stat(configPath)
			if err == nil {
				// A config file exists, use it
				configFile, err := os.ReadFile(configPath)
				require.NoError(t, err)

				err = config.Analyzer.Flags.Set(config.YamlConfig, string(configFile))
				require.NoError(t, err)
			} else if !os.IsNotExist(err) {
				require.FailNow(t, err.Error())
			}

			r := analysistest.Run(t, testdata, errstack.Analyzer, f.Name())
			res := r[0].Result

			result := res.(*helpers.Result[*errstack.Result])
			require.NoError(t, result.Err)
		})
	}
}

func TestConfig(t *testing.T) {
	pass := &analysis.Pass{Analyzer: config.Analyzer}
	res, err := config.Analyzer.Run(pass)
	require.NoError(t, err)

	_ = config.Analyzer.Flags.Set(config.Debug, "true")
	t.Cleanup(func() {
		_ = config.Analyzer.Flags.Set(config.Debug, "")
	})

	result := res.(*helpers.Result[*config.Config])
	require.NoError(t, result.Err)
	conf := result.Res

	require.ElementsMatch(t, conf.WrapperFunctions, config.DefaultWrapperFunctions)
	require.ElementsMatch(t, conf.CleanFunctions, config.DefaultCleanFunctions)
}

func TestSingle(t *testing.T) {
	testdata := analysistest.TestData()
	chdir(t, testdata+"/src")

	_ = config.Analyzer.Flags.Set(config.Debug, "true")
	t.Cleanup(func() {
		_ = config.Analyzer.Flags.Set(config.Debug, "")
	})

	r := analysistest.Run(t, testdata, errstack.Analyzer, "cfgs_branches")
	require.GreaterOrEqual(t, len(r), 1)
	res := r[0].Result

	result := res.(*helpers.Result[*errstack.Result])
	require.NoError(t, result.Err)
}

func TestAnalyzerPerformance(t *testing.T) {
	testdata := analysistest.TestData()

	// Test with a time limit to catch performance issues
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		chdir(t, testdata+"/src")
		analysistest.Run(t, testdata, errstack.Analyzer, "builtins")
	}()

	select {
	case <-done:
		// Test completed successfully
	case <-ctx.Done():
		t.Fatal("Test timed out - performance issue detected")
	}
}

func TestFalsePositives(t *testing.T) {
	// Test cases that should NOT trigger excessive warnings
	testCases := []string{
		"interface",    // Interface implementations
		"external_pkg", // External package usage
		"method",       // Method calls
	}

	testdata := analysistest.TestData()
	chdir(t, testdata+"/src")

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			// Run analyzer and check for unexpected reports
			result := analysistest.Run(t, testdata, errstack.Analyzer, tc)
			require.GreaterOrEqual(t, len(result), 1)

			res := result[0].Result
			errResult := res.(*helpers.Result[*errstack.Result])
			require.NoError(t, errResult.Err)
		})
	}
}
