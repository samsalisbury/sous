package sous

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/samsalisbury/semv"
)

type (
	// RegistryDumper dumps the contents of artifact registries
	RegistryDumper struct {
		Registry
		log logging.LogSink
	}

	// DumperEntry is a single entry from the dump
	DumperEntry struct {
		SourceID
		*BuildArtifact
	}
)

// NewRegistryDumper constructs a RegistryDumper
func NewRegistryDumper(r Registry, ls logging.LogSink) *RegistryDumper {
	return &RegistryDumper{Registry: r, log: ls}
}

// AsTable writes a tabular dump of the registry to a Writer
func (rd *RegistryDumper) AsTable(to io.Writer) error {
	w := &tabwriter.Writer{}
	w.Init(to, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, rd.TabbedHeaders())

	es, err := rd.Entries()
	if err != nil {
		return err
	}
	for _, e := range es {
		fmt.Fprintln(w, e.Tabbed())
	}
	w.Flush()
	return nil
}

// TabbedHeaders outputs the headers for the dump
func (rd *RegistryDumper) TabbedHeaders() string {
	return "Repo\tOffset\tVersion\tName\tType"
}

// Entries emits the list of entries for the Resgistry
func (rd *RegistryDumper) Entries() (de []DumperEntry, err error) {
	ss, err := rd.Registry.ListSourceIDs()
	messages.ReportLogFieldsMessage("List Source IDS", logging.ExtraDebug1Level, rd.log, ss)
	if err != nil {
		return
	}

	for _, s := range ss {
		a, err := rd.Registry.GetArtifact(s)
		if err != nil {
			return nil, err
		}
		messages.ReportLogFieldsMessage("Source Id and Artifact", logging.ExtraDebug1Level, rd.log, s, a)
		de = append(de, DumperEntry{SourceID: s, BuildArtifact: a})
	}

	return
}

// Tabbed emits a tab-delimited string representing the entry
func (de *DumperEntry) Tabbed() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s", de.Location.Repo, de.Location.Dir, de.Version.Format(semv.MajorMinorPatch), de.DigestReference, de.Type)
}
