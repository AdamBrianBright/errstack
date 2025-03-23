package preload_packages

import (
	"go/token"
	"reflect"
	"sync"

	"github.com/AdamBrianBright/errstack/internal/config"
	"github.com/AdamBrianBright/errstack/internal/helpers"
	"github.com/AdamBrianBright/errstack/internal/log"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

const _doc = `Preloads all packages and parses their ASTs.`

var Analyzer = &analysis.Analyzer{
	Name:       "errstack_preload_packages",
	Doc:        _doc,
	Run:        helpers.WrapRun(run),
	ResultType: reflect.TypeOf((*helpers.Result[*Result])(nil)),
	Requires:   []*analysis.Analyzer{config.Analyzer},
}

var done chan struct{}
var once sync.Once
var result = &Result{
	Pkgs: map[string]*packages.Package{},
	Objs: map[token.Position]NodeInfo{},
}

func init() {
	done = make(chan struct{})
}

func run(pass *analysis.Pass) (*Result, error) {
	log.Log("Preloading packages\n")

	go once.Do(func() {
		defer close(done)
		log.Log("Preloading packages once\n")

		conf, _ := helpers.GetResult[*config.Config](pass, config.Analyzer)
		result.conf = conf

		pkgs, err := packages.Load(&packages.Config{
			Mode: packages.LoadAllSyntax,
		}, result.conf.GoRoot+"/...", result.conf.WorkDir+"/...", result.conf.WorkDir+"vendor/...")
		if err != nil {
			panic(err)
		}
		for _, pkg := range pkgs {
			result.Pkgs[result.conf.GetDirPkgPath(pkg.Dir)] = pkg
		}
	})
	<-done

	return result, nil
}
