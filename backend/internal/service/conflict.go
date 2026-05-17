package service

import (
	"fmt"
	"path/filepath"

	"github.com/honeok/honepress/internal/model"
)

type permalinkOwner struct {
	sourceFilePath string
}

func validatePermalinkConflicts(posts []model.Post) error {
	permalinkOwners := make(map[string]permalinkOwner)
	for _, currentPost := range posts {
		if err := claimPermalink(permalinkOwners, currentPost.URL, currentPost.SourceFilePath); err != nil {
			return err
		}
		if currentPost.SourceURL != "" && currentPost.SourceURL != currentPost.URL {
			if err := claimPermalink(permalinkOwners, currentPost.SourceURL, currentPost.SourceFilePath); err != nil {
				return err
			}
		}
	}
	return nil
}

func claimPermalink(permalinkOwners map[string]permalinkOwner, permalink string, sourceFilePath string) error {
	if previousOwner, exists := permalinkOwners[permalink]; exists {
		return fmt.Errorf("permanent URL conflict: %s is used by both %s and %s", permalink, displaySourcePath(previousOwner.sourceFilePath), displaySourcePath(sourceFilePath))
	}
	permalinkOwners[permalink] = permalinkOwner{sourceFilePath: sourceFilePath}
	return nil
}

func displaySourcePath(sourceFilePath string) string {
	return filepath.ToSlash(sourceFilePath)
}
