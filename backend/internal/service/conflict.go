package service

import (
	"fmt"
	"path/filepath"

	"github.com/honeok/honepress/internal/model"
)

type permalinkOwner struct {
	sourceFilePath string
	isAlias        bool
}

func validatePermalinkConflicts(posts []model.Post) error {
	permalinkOwners := make(map[string]permalinkOwner)
	for _, currentPost := range posts {
		if err := claimPermalink(permalinkOwners, currentPost.URL, currentPost.SourceFilePath, false); err != nil {
			return err
		}
		if currentPost.SourceURL != "" && currentPost.SourceURL != currentPost.URL {
			if err := claimPermalink(permalinkOwners, currentPost.SourceURL, currentPost.SourceFilePath, true); err != nil {
				return err
			}
		}

		aliasOwners := make(map[string]struct{})
		for _, currentAlias := range currentPost.Aliases {
			if _, exists := aliasOwners[currentAlias]; exists {
				return fmt.Errorf("alias conflict: %s is duplicated in %s", currentAlias, displaySourcePath(currentPost.SourceFilePath))
			}
			aliasOwners[currentAlias] = struct{}{}

			if previousOwner, exists := permalinkOwners[currentAlias]; exists {
				if previousOwner.isAlias {
					return fmt.Errorf("alias conflict: %s is used by both %s and %s", currentAlias, displaySourcePath(previousOwner.sourceFilePath), displaySourcePath(currentPost.SourceFilePath))
				}
				return fmt.Errorf("alias conflict: %s is already used as a permalink by %s", currentAlias, displaySourcePath(previousOwner.sourceFilePath))
			}
			permalinkOwners[currentAlias] = permalinkOwner{sourceFilePath: currentPost.SourceFilePath, isAlias: true}
		}
	}
	return nil
}

func claimPermalink(permalinkOwners map[string]permalinkOwner, permalink string, sourceFilePath string, isAlias bool) error {
	if previousOwner, exists := permalinkOwners[permalink]; exists {
		if isAlias || previousOwner.isAlias {
			return fmt.Errorf("alias conflict: %s is used by both %s and %s", permalink, displaySourcePath(previousOwner.sourceFilePath), displaySourcePath(sourceFilePath))
		}
		return fmt.Errorf("permalink conflict: %s is used by both %s and %s", permalink, displaySourcePath(previousOwner.sourceFilePath), displaySourcePath(sourceFilePath))
	}
	permalinkOwners[permalink] = permalinkOwner{sourceFilePath: sourceFilePath, isAlias: isAlias}
	return nil
}

func displaySourcePath(sourceFilePath string) string {
	return filepath.ToSlash(sourceFilePath)
}
