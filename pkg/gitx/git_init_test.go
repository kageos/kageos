package gitx

import (
	"fmt"
	git2 "github.com/go-git/go-git/v5"
	"github.com/yunhanshu-net/pkg/x/filex"
	"github.com/yunhanshu-net/pkg/x/jsonx"
	"testing"
)

type Commit struct {
	Version string `json:"version"`
	Desc    string `json:"desc"`
}

func TestGitInit(t *testing.T) {
	filex.MustCreateFileAndWriteContent("./testgit/v3.txt", "v3")
	git, err := InitOrOpen("./testgit", "beiluo", "beiluo@test.com")
	if err != nil {
		panic(err)
	}
	err = git.Add("./")
	if err != nil {
		panic(err)
	}
	commit, err := git.Commit(jsonx.JSONString(&Commit{Version: "v3", Desc: "测试"}))
	if err != nil {
		panic(err)
	}
	fmt.Println(commit)

}

func TestReset(t *testing.T) {
	//filex.MustCreateFileAndWriteContent("./testgit/v3.txt", "v3")
	git, err := InitOrOpen("./testgit", "beiluo", "beiluo@test.com")
	if err != nil {
		panic(err)
	}
	err = git.ResetByJSONField("version", "v2", git2.HardReset)
	if err != nil {
		panic(err)
	}

}

func TestCreateBatch(t *testing.T) {
	git, err := InitOrOpen("./testgit", "beiluo", "beiluo@test.com")
	if err != nil {
		panic(err)
	}
	err = git.CreateBranchAndCheckout("v3")
	if err != nil {
		panic(err)
	}

}
