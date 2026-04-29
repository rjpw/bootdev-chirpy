package chirpy_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

type Package struct {
	Name       string   `json:"Name"`
	ImportPath string   `json:"ImportPath"`
	Imports    []string `json:"Imports"`
}

var (
	roles map[string]string
	rules map[string][]string
)

func TestMain(m *testing.M) {
	for _, f := range []struct {
		path string
		dest any
	}{
		{"testdata/hex_roles.json", &roles},
		{"testdata/hex_rules.json", &rules},
	} {
		data, err := os.ReadFile(f.path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "loading %s: %v\n", f.path, err)
			os.Exit(1)
		}
		if err := json.Unmarshal(data, f.dest); err != nil {
			fmt.Fprintf(os.Stderr, "parsing %s: %v\n", f.path, err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

func resolveRoleKey(longName string) string {
	parts := strings.Split(longName, "/")
	pkgName := strings.Join(parts[3:5], "/")
	return pkgName
}

func getParentPackageRole(p Package) string {
	return roles[resolveRoleKey(p.ImportPath)]
}

func applyRules(t *testing.T, p Package) {
	/*
		The rules:
			- core → no internal imports allowed
			- driving → only core and root
			- driven → only core and own sub-packages
			- infra → no core, driving, or driven
			- root → unrestricted
	*/
	t.Helper()
	packageName := resolveRoleKey(p.ImportPath)
	parentPackageRole := getParentPackageRole(p)
	t.Logf("PackageRole: %s\n", parentPackageRole)

	var localImportPackages []string

	for _, s := range p.Imports {
		if strings.HasPrefix(s, "github.com/rjpw/bootdev-chirpy") {
			localImportPackages = append(localImportPackages, s)
		}
	}

	allowedRoles := rules[parentPackageRole]
	for _, importPackage := range localImportPackages {
		localPackageName := resolveRoleKey(importPackage)
		localPackageRole := getParentPackageRole(Package{ImportPath: importPackage})
		if localPackageName != packageName { // note: we assume sub-modules are safe to call
			t.Logf("Checking if %s can call %s\n", packageName, localPackageName)
			if !slices.Contains(allowedRoles, localPackageRole) {
				t.Errorf(
					"hex violation: %s (%s) imports %s (%s)",
					packageName,
					parentPackageRole,
					localPackageName,
					localPackageRole,
				)
			}
		}
	}
}

func TestHexBoundaries(t *testing.T) {
	// fmt.Printf("%v\n", roles)
	cmd := exec.Command("go", "list", "-json", "./internal/...")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("...: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("...: %v", err)
	}

	decoder := json.NewDecoder(stdout)

	for decoder.More() {
		var p Package
		err := decoder.Decode(&p)
		if err != nil {
			t.Logf("Error: %v\n", err)
			break
		}
		t.Logf("Package: %+v\n", p.ImportPath)
		applyRules(t, p)
	}

	cmd.Wait()
}
