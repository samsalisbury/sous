package actions

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/git"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/shell"
	"github.com/samsalisbury/semv"
)

// A Server represents the `sous server` command.
type Server struct {
	Version           semv.Version
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	Log               logging.LogSink

	ListenAddr string
	GDMRepo    string

	*config.Config
	ServerHandler http.Handler
	*sous.AutoResolver
}

// Do runs the server.
func (ss *Server) Do() error {
	if err := ensureGDMExists(ss.GDMRepo, ss.Config.StateLocation, ss.DeployFilterFlags, ss.ListenAddr, ss.Log); err != nil {
		return err
	}

	reportServerMessage("Starting scheduled GDM resolution.  Filtering the GDM to resolve on this server", ss.DeployFilterFlags, ss.ListenAddr, ss.Log)

	ss.AutoResolver.Kickoff()

	reportServerMessage("Sous Server Running", ss.DeployFilterFlags, ss.ListenAddr, ss.Log)

	return server.Run(ss.ListenAddr, ss.ServerHandler)
}

func ensureGDMExists(repo, localPath string, filterFlags config.DeployFilterFlags, listenAddress string, log logging.LogSink) error {
	s, err := os.Stat(localPath)
	if err == nil && s.IsDir() {
		files, err := ioutil.ReadDir(localPath)
		if err != nil {
			return err
		}
		if len(files) != 0 {
			// The directory exists and is not empty, do nothing.
			if repo != "" {
				msg := fmt.Sprintf("not pulling repo %q; directory already exist and is not empty: %q", repo, localPath)
				reportServerMessage(msg, filterFlags, listenAddress, log)
			}
			return nil
		}
	}
	if err := config.EnsureDirExists(localPath); err != nil {
		return err
	}
	// xxx Shouldn't this simply fail if there's no GDM available?
	sh, err := shell.DefaultInDir(localPath)
	if err != nil {
		return err
	}
	g, err := git.NewClient(sh)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("cloning %q into %q ...", repo, localPath)
	reportServerMessage(msg, filterFlags, listenAddress, log)

	if err := g.CloneRepo(repo, localPath); err != nil {
		return err
	}

	reportServerMessage("done", filterFlags, listenAddress, log)

	return nil
}

type serverMessage struct {
	logging.CallerInfo
	msg               string
	deployFilterFlags config.DeployFilterFlags
	listenAddress     string
}

func reportServerMessage(msg string, filterFlags config.DeployFilterFlags, listenAddress string, log logging.LogSink) {
	msgLog := serverMessage{
		msg:               msg,
		CallerInfo:        logging.GetCallerInfo(logging.NotHere()),
		deployFilterFlags: filterFlags,
		listenAddress:     listenAddress,
	}
	logging.Deliver(msgLog, log)
}

func (msg serverMessage) WriteToConsole(console io.Writer) {
	fmt.Fprintf(console, "%s\n", msg.composeMsg())
}

func (msg serverMessage) DefaultLevel() logging.Level {
	return logging.WarningLevel
}

func (msg serverMessage) Message() string {
	return msg.composeMsg()
}

func (msg serverMessage) composeMsg() string {
	return fmt.Sprintf("%s, server at %s for %s: DeployFilter Flags %v", msg.msg, msg.listenAddress, msg.deployFilterFlags.Cluster, msg.deployFilterFlags)
}

func (msg serverMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-generic-v1")
	f("listenAddress", msg.listenAddress)
	msg.CallerInfo.EachField(f)
}
