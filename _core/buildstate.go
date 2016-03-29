package core

import (
	"crypto/sha1"
	"fmt"
	"io"

	"github.com/opentable/sous/tools/dir"
	"github.com/opentable/sous/tools/file"
	"github.com/opentable/sous_old/tools/cmd"
	"github.com/opentable/sous_old/tools/git"
)

type BuildState struct {
	CommitSHA, LastCommitSHA string
	Commits                  map[string]*Commit
	path                     string
}

type Commit struct {
	Hash, OldHash         string
	TreeHash, OldTreeHash string
	SousHash, OldSousHash string
	BuildNumber           int
	ToolVersion           string
}

func GetBuildState(action string, g *git.Info) (*BuildState, error) {
	filePath := getStateFile(action, g)
	var state *BuildState
	if !file.ReadJSON(&state, filePath) {
		state = &BuildState{
			Commits: map[string]*Commit{},
		}
	}
	if state == nil {
		return fmt.Errorf("Nil state at %s", filePath)
	}
	if state.Commits == nil {
		return fmt.Errorf("Nil commits at %s", filePath)
	}
	c, ok := state.Commits[g.CommitSHA]
	if !ok {
		state.Commits[g.CommitSHA] = &Commit{}
	}
	state.LastCommitSHA = state.CommitSHA
	state.CommitSHA = g.CommitSHA
	state.path = filePath
	c = state.Commits[g.CommitSHA]
	if buildingInCI() {
		bn, ok := tryGetBuildNumberFromEnv()
		if !ok {
			return fmt.Errorf("unable to get build number from $BUILD_NUMBER TeamCity")
		}
		c.BuildNumber = bn
	}
	c.OldTreeHash = c.TreeHash
	c.TreeHash = CalculateTreeHash()
	c.OldSousHash = c.SousHash
	c.SousHash = CalculateSousHash()
	c.OldHash = c.Hash
	c.Hash = HashSum(c.TreeHash, c.SousHash)
	return state
}

// HashSum(inputs ...string) returns a hash of all the input strings concatenated
func HashSum(inputs ...string) string {
	h := sha1.New()
	for _, i := range inputs {
		io.WriteString(h, i)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// CalculateTreeHash returns a hash of the current state of the working tree,
// leaning heavily on git for optimisation.
func CalculateTreeHash() string {
	inputs := []string{}
	indexDiffs := cmd.Stdout("git", "diff-index", "HEAD")
	if len(indexDiffs) != 0 {
		inputs = append(inputs, indexDiffs)
	}
	newFiles := git.UntrackedUnignoredFiles()
	if len(newFiles) != 0 {
		for _, f := range newFiles {
			inputs = append(inputs, f)
			if content, ok := file.ReadString(f); ok {
				inputs = append(inputs, content)
			}
		}
	}
	return HashSum(inputs...)
}

// CalculateSousHash returns a hash of the current version of Sous and its main
// configuration file.
func CalculateSousHash() string {
	inputs := []string{cmd.Stdout("sous", "version")}
	if c, ok := file.ReadString("~/.sous/config"); ok {
		inputs = append(inputs, c)
	}
	return HashSum(inputs...)
}

func getStateFile(action string, g *git.Info) string {
	dirPath := fmt.Sprintf("~/.sous/builds/%s/%s", g.CanonicalRepoName(), action)
	dir.EnsureExists(dirPath)
	return fmt.Sprintf("%s/state", dirPath)
}
