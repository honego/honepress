package service

import (
	"fmt"
	"path/filepath"

	"github.com/honeok/blog/model"
)

type permalinkOwner struct {
	sourceFilePath string
	isAlias        bool
}

func validatePermalinkConflicts(posts []model.Post) error {
	permalinkOwners := make(map[string]permalinkOwner)
	for _, currentPost := range posts {
		if previousOwner, exists := permalinkOwners[currentPost.URL]; exists {
			return fmt.Errorf("固定链接冲突：%s 同时被 %s 和 %s 使用", currentPost.URL, displaySourcePath(previousOwner.sourceFilePath), displaySourcePath(currentPost.SourceFilePath))
		}
		permalinkOwners[currentPost.URL] = permalinkOwner{sourceFilePath: currentPost.SourceFilePath}

		aliasOwners := make(map[string]struct{})
		for _, currentAlias := range currentPost.Aliases {
			if _, exists := aliasOwners[currentAlias]; exists {
				return fmt.Errorf("别名链接冲突：%s 在 %s 中重复使用", currentAlias, displaySourcePath(currentPost.SourceFilePath))
			}
			aliasOwners[currentAlias] = struct{}{}

			if previousOwner, exists := permalinkOwners[currentAlias]; exists {
				if previousOwner.isAlias {
					return fmt.Errorf("别名链接冲突：%s 同时被 %s 和 %s 使用", currentAlias, displaySourcePath(previousOwner.sourceFilePath), displaySourcePath(currentPost.SourceFilePath))
				}
				return fmt.Errorf("别名链接冲突：%s 已被 %s 作为固定链接使用", currentAlias, displaySourcePath(previousOwner.sourceFilePath))
			}
			permalinkOwners[currentAlias] = permalinkOwner{sourceFilePath: currentPost.SourceFilePath, isAlias: true}
		}
	}
	return nil
}

func displaySourcePath(sourceFilePath string) string {
	return filepath.ToSlash(sourceFilePath)
}
