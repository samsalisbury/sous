package core

import (
	"fmt"
	"os"

	"github.com/opentable/sous/tools/docker"
)

func CheckForProblems(pack Pack) (fatal bool) {
	// Now we know that the user was asking for something possible with the detected build pack,
	// let's make sure that build pack is properly compatible with this project
	issues := pack.Problems()
	warnings, errors := issues.Warnings(), issues.Errors()
	if len(warnings) != 0 {
		// TODO: Emit warnings
		//cli.LogBulletList("WARNING:", issues.Strings())
	}
	if len(errors) != 0 {
		// TODO: Emit errors
		//cli.LogBulletList("ERROR:", errors.Strings())
		//cli.Logf("ERROR: Your project cannot be built by Sous until the above errors are rectified")
		return true
	}
	return false
}

func (s *Sous) TargetContext(targetName string) (*TargetContext, error) {
	context := GetContext()
	pack := context.DetectProjectType(s.State.Buildpacks)
	if pack == nil {
		return fmt.Errorf("no buildable project detected")
	}
	runnablePack, err := pack.BindStackVersion(context.WorkDir)
	if err != nil {
		return err
	}
	target := GetTarget(runnablePack, context, targetName)
	if err := target.Check(); err != nil {
		return fmt.Errorf("unable to %s %s project: %s", targetName, pack, err)
	}
	bs := GetBuildState(targetName, context.Git)
	return &TargetContext{
		TargetName: targetName,
		BuildState: bs,
		Buildpack:  runnablePack,
		Context:    context,
		Target:     target,
	}
}

func GetTarget(bp *RunnableBuildpack, c *Context, name string) Target {
	switch name {
	default:
		return nil
	case "app":
		return NewAppTarget(bp, c)
	case "compile":
		return NewCompileTarget(bp, c)
	case "test":
		return nil
		//return NewTestTarget(bp)
	}
}

func DivineTaskHost() string {
	taskHost := os.Getenv("TASK_HOST")
	if taskHost != "" {
		return taskHost
	}
	return docker.GetDockerHost()
}
