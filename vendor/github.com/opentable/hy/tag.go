package hy

import (
	"strings"

	"github.com/pkg/errors"
)

// Tag contains a parsed struct field tag.
type Tag struct {
	None,
	Ignore,
	IsDir bool
	PathName,
	Key,
	SetKey string
}

func parseTag(tagString string) (Tag, error) {
	if tagString == "" {
		return Tag{None: true}, nil
	}
	var pathName, key, setKey string
	parts := strings.Split(tagString, ",")
	if len(parts) > 0 {
		pathName = parts[0]
	}
	if len(parts) > 1 {
		key = parts[1]
	}
	if len(parts) > 2 {
		setKey = parts[2]
	}
	if len(parts) > 3 {
		return Tag{}, errors.Errorf("malformed tag, too many commas")
	}
	if pathName == "-" {
		return Tag{Ignore: true}, nil
	}
	pathName, isDir, err := parsePathName(pathName)
	if err != nil {
		return Tag{}, errors.Wrapf(err, "path name %q invalid", pathName)
	}
	return Tag{
		IsDir:    isDir,
		PathName: pathName,
		Key:      key,
		SetKey:   setKey,
	}, nil
}

func parsePathName(pathName string) (string, bool, error) {
	if pathName == "" {
		return ".", false, nil
	}
	if pathName == "/" {
		return ".", true, nil
	}
	if strings.HasPrefix(pathName, "/") {
		return pathName, false, errors.Errorf("must not begin with /")
	}
	pathNameSuffix := pathName[len(pathName)-1:]
	switch pathNameSuffix {
	default:
		return pathName, false, nil
	case "/":
		return strings.TrimSuffix(pathName, "/"), true, nil
	}
}
