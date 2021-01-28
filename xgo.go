// Wrapper around the GCO cross compiler docker container.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var version = "dev"
var depsCache = filepath.Join(os.TempDir(), "xgo-cache")

// Cross compilation docker containers
var dockerDist = "crazymax/xgo"

// Command line arguments to fine tune the compilation
var (
	goVersion   = flag.String("go", "latest", "Go release to use for cross compilation")
	srcPackage  = flag.String("pkg", "", "Sub-package to build if not root import")
	srcRemote   = flag.String("remote", "", "Version control remote repository to build")
	srcBranch   = flag.String("branch", "", "Version control branch to build")
	outPrefix   = flag.String("out", "", "Prefix to use for output naming (empty = package name)")
	outFolder   = flag.String("dest", "", "Destination folder to put binaries in (empty = current)")
	crossDeps   = flag.String("deps", "", "CGO dependencies (configure/make based archives)")
	crossArgs   = flag.String("depsargs", "", "CGO dependency configure arguments")
	targets     = flag.String("targets", "*/*", "Comma separated targets to build for")
	dockerRepo  = flag.String("docker-repo", "", "Use custom docker repo instead of official distribution")
	dockerImage = flag.String("docker-image", "", "Use custom docker image instead of official distribution")
	goMod       = flag.String("mod", "false", "go mod:use go mod or not")
	buildDir    = flag.String("buildDir", "", "build dir file wor with pkg:sub build dir")
	goPath      = flag.String("goPath", "", "go mod:set the go path")
	goProxy     = flag.String("goproxy", "", "go mod:Set a Global Proxy for Go Modules")
)

// ConfigFlags is a simple set of flags to define the environment and dependencies.
type ConfigFlags struct {
	Repository   string   // relative dir
	Package      string   // Sub-package to build if not root import
	Prefix       string   // Prefix to use for output naming
	Remote       string   // Version control remote repository to build
	Branch       string   // Version control branch to build
	Dependencies string   // CGO dependencies (configure/make based archives)
	Arguments    string   // CGO dependency configure arguments
	GoPath       string   // go path
	BuildDir     string   // build dir arguments
	Targets      []string // Targets to build for
}

// Command line arguments to pass to go build
var (
	buildVerbose = flag.Bool("v", false, "Print the names of packages as they are compiled")
	buildSteps   = flag.Bool("x", false, "Print the command as executing the builds")
	buildRace    = flag.Bool("race", false, "Enable data race detection (supported only on amd64)")
	buildTags    = flag.String("tags", "", "List of build tags to consider satisfied during the build")
	buildLdFlags = flag.String("ldflags", "", "Arguments to pass on each go tool link invocation")
	buildMode    = flag.String("buildmode", "default", "Indicates which kind of object file to build")
)

// BuildFlags is a simple collection of flags to fine tune a build.
type BuildFlags struct {
	Verbose bool   // Print the names of packages as they are compiled
	Steps   bool   // Print the command as executing the builds
	Race    bool   // Enable data race detection (supported only on amd64)
	Tags    string // List of build tags to consider satisfied during the build
	LdFlags string // Arguments to pass on each go tool link invocation
	Mode    string // Indicates which kind of object file to build
}

func main() {
	defer fmt.Println("🏁 Finished!")
	var err error
	image := "" // Only use docker images if we're not already inside out own image

	fmt.Printf("🚀 Start xgo %s\n", version)

	// Retrieve the CLI flags and the execution environment
	flag.Parse()

	if *buildDir == "" {
		log.Fatal("❌ missing build dir")
	}

	if len(flag.Args()) != 1 {
		log.Fatalf("👉 Usage: %s [options] <go import path>", os.Args[0])
	}

	// Ensure docker is available
	if err := checkDocker(); err != nil {
		log.Fatalf("❌ Failed to check docker installation: %v.", err)
	}

	// Select the image to use, either official or custom

	image = fmt.Sprintf("%s:%s", dockerDist, *goVersion)

	if *dockerImage != "" {
		image = *dockerImage
	} else if *dockerRepo != "" {
		image = fmt.Sprintf("%s:%s", *dockerRepo, *goVersion)
	}

	// Check that all required images are available
	found, err := checkDockerImage(image)
	switch {
	case err != nil:
		log.Fatalf("❌ Failed to check docker image availability: %v.", err)
	case !found:
		fmt.Println("not found!")
		if err := pullDockerImage(image); err != nil {
			log.Fatalf("❌ Failed to pull docker image from the registry: %v.", err)
		}
	default:
		fmt.Println("🎉 Docker image found!")
	}

	// Cache all external dependencies to prevent always hitting the internet
	if *crossDeps != "" {
		if err := os.MkdirAll(depsCache, 0751); err != nil {
			log.Fatalf("❌ Failed to create dependency cache: %v.", err)
		}
		// Download all missing dependencies
		for _, dep := range strings.Split(*crossDeps, " ") {
			if url := strings.TrimSpace(dep); len(url) > 0 {
				path := filepath.Join(depsCache, filepath.Base(url))

				if _, err := os.Stat(path); err != nil {
					fmt.Printf("⬇️ Downloading new dependency: %s...\n", url)

					out, err := os.Create(path)
					if err != nil {
						log.Fatalf("❌ Failed to create dependency file: %v.", err)
					}
					res, err := http.Get(url)
					if err != nil {
						log.Fatalf("❌ Failed to retrieve dependency: %v.", err)
					}
					defer res.Body.Close()

					if _, err := io.Copy(out, res.Body); err != nil {
						log.Fatalf("❌ Failed to download dependency: %v", err)
					}
					out.Close()

					fmt.Printf("✨ New dependency cached: %s.\n", path)
				} else {
					fmt.Printf("🤝 Dependency already cached: %s.\n", path)
				}
			}
		}
	}

	// Assemble the cross compilation environment and build options
	config := &ConfigFlags{
		Repository:   flag.Args()[0],
		Package:      *srcPackage,
		Remote:       *srcRemote,
		Branch:       *srcBranch,
		Prefix:       *outPrefix,
		Dependencies: *crossDeps,
		Arguments:    *crossArgs,
		GoPath:       *goPath,
		BuildDir:     *buildDir,
		Targets:      strings.Split(*targets, ","),
	}

	flags := &BuildFlags{
		Verbose: *buildVerbose,
		Steps:   *buildSteps,
		Race:    *buildRace,
		Tags:    *buildTags,
		LdFlags: *buildLdFlags,
		Mode:    *buildMode,
	}

	folder := config.BuildDir

	// Execute the cross compilation, either in a container or the current system
	err = compile(image, config, flags, folder)
	if err != nil {
		log.Fatalf("❌ Failed to cross compile package: %v.", err)
	}
}

// Checks whether a docker installation can be found and is functional.
func checkDocker() error {
	fmt.Println("🐳 Checking docker installation...")
	if err := run(exec.Command("sh", "-c", fmt.Sprintf(`docker version |grep 'Server'`))); err != nil {
		return err
	}

	return nil
}

// Checks whether a required docker image is available locally.
func checkDockerImage(image string) (bool, error) {
	fmt.Printf("🐳 Checking for required docker image %s... ", image)
	out, err := exec.Command("sh", "-c", fmt.Sprintf(`docker images --no-trunc | awk '{print $1":"$2}' | grep '%s'`, image)).Output()
	if err != nil {
		return false, err
	}

	return bytes.Contains(out, []byte(image)), nil
}

// Pulls an image from the docker registry.
func pullDockerImage(image string) error {
	fmt.Printf("🐳 Pulling %s from docker registry...\n", image)
	return run(exec.Command("docker", "pull", image))
}

// compile cross builds a requested package according to the given build specs
// using a specific docker cross compilation image.
func compile(image string, config *ConfigFlags, flags *BuildFlags, folder string) (err error) {
	var usesModules bool

	if *goMod == "true" {
		usesModules = true

		var modFile = folder + "/go.mod"
		_, err := os.Stat(modFile)
		if err != nil {
			log.Fatal("❌ Failed stat mod file:", err)
		}
	}

	// Assemble and run the cross compilation command
	fmt.Printf("🏃 Cross compiling %s... \n", config.Repository)

	args := []string{
		"run", "--rm",
		"-v", folder + ":/build",
		"-v", depsCache + ":/deps-cache:ro",
		"-e", "REPO_REMOTE=" + config.Remote,
		"-e", "REPO_BRANCH=" + config.Branch,
		"-e", "PACK=" + config.Package,
		"-e", "DEPS=" + config.Dependencies,
		"-e", "ARGS=" + config.Arguments,
		"-e", "OUT=" + config.Prefix,
		"-e", fmt.Sprintf("FLAG_V=%v", flags.Verbose),
		"-e", fmt.Sprintf("FLAG_X=%v", flags.Steps),
		"-e", fmt.Sprintf("FLAG_RACE=%v", flags.Race),
		"-e", fmt.Sprintf("FLAG_TAGS=%s", flags.Tags),
		"-e", fmt.Sprintf("FLAG_LDFLAGS=%s", flags.LdFlags),
		"-e", fmt.Sprintf("FLAG_BUILDMODE=%s", flags.Mode),
		"-e", "TARGETS=" + strings.Replace(strings.Join(config.Targets, " "), "*", ".", -1),
	}

	if *goProxy != "" {
		args = append(args, []string{"-e", fmt.Sprintf("GOPROXY=%s", *goProxy)}...)
	}

	//set cache dir
	if config.GoPath != "" {
		args = append(args, []string{"-v", config.GoPath + ":/cache"}...)
		args = append(args, []string{"-e", "GOPATH=/cache"}...)
	}

	if usesModules {
		args = append(args, []string{"-e", "GO111MODULE=on"}...)

		args = append(args, []string{"-v", folder + ":/source"}...)

		fmt.Printf("✅ Enabled Go module support\n")

		// Check whether it has a vendor folder, and if so, use it
		vendorPath := config.Repository + "/vendor"
		vendorfolder, err := os.Stat(vendorPath)
		if !os.IsNotExist(err) && vendorfolder.Mode().IsDir() {
			args = append(args, []string{"-e", "FLAG_MOD=vendor"}...)
			fmt.Printf("✅ Using vendored Go module dependencies\n")
		}
	}

	args = append(args, []string{image, config.Repository}...)
	fmt.Printf("🐳 Docker %s\n", strings.Join(args, " "))
	return run(exec.Command("docker", args...))
}

// resolveImportPath converts a package given by a relative path to a Go import
// path using the local GOPATH environment.
func resolveImportPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("❌ Failed to locate requested package: %v.", err)
	}
	stat, err := os.Stat(abs)
	if err != nil || !stat.IsDir() {
		log.Fatalf("❌ Requested path invalid.")
	}
	pack, err := build.ImportDir(abs, build.FindOnly)
	if err != nil {
		log.Fatalf("❌ Failed to resolve import path: %v.", err)
	}
	return pack.ImportPath
}

// Executes a command synchronously, redirecting its output to stdout.
func run(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
