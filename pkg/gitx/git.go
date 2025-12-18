package gitx

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GitProject 表示一个git项目
type GitProject struct {
	repo     *git.Repository
	worktree *git.Worktree
	path     string
	author   *object.Signature
}

// BranchInfo 分支信息
type BranchInfo struct {
	Name   string
	Hash   string
	IsHead bool
}

// FileStatus 文件状态
type FileStatus struct {
	Path     string
	Worktree git.StatusCode
	Staging  git.StatusCode
}

// CommitInfo 提交信息
type CommitInfo struct {
	Hash    string
	Message string
	Author  string
	Time    time.Time
}

// CommitMatcher 提交匹配器接口
type CommitMatcher interface {
	Match(commit *object.Commit) (bool, error)
}

// CommitMatcherFunc 提交匹配器函数类型
type CommitMatcherFunc func(commit *object.Commit) bool

// Match 实现 CommitMatcher 接口
func (f CommitMatcherFunc) Match(commit *object.Commit) bool {
	return f(commit)
}

// CommitCompareFunc 提交比较函数类型
type CommitCompareFunc func(commit *object.Commit, resetCommitMsg string) bool

// NewGitProject 创建一个新的GitProject实例
func NewGitProject(path string, authorName, authorEmail string) (*GitProject, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("打开仓库失败: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("获取工作树失败: %v", err)
	}

	return &GitProject{
		repo:     repo,
		worktree: worktree,
		path:     path,
		author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	}, nil
}

// InitOrOpen 初始化或打开仓库
// 如果目录不存在，则创建目录并初始化仓库
// 如果目录存在但不是git仓库，则初始化仓库
// 如果目录存在且是git仓库，则打开仓库
func InitOrOpen(path string, authorName, authorEmail string) (*GitProject, error) {
	// 检查目录是否存在
	exists, err := existsFile(path)
	if err != nil {
		return nil, fmt.Errorf("检查目录是否存在失败: %v", err)
	}

	// 如果目录不存在，创建目录
	if !exists {
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("创建目录失败: %v", err)
		}
	}

	// 尝试打开仓库
	repo, err := git.PlainOpen(path)
	if err == nil {
		// 仓库存在，获取工作树
		worktree, err := repo.Worktree()
		if err != nil {
			return nil, fmt.Errorf("获取工作树失败: %v", err)
		}

		return &GitProject{
			repo:     repo,
			worktree: worktree,
			path:     path,
			author: &object.Signature{
				Name:  authorName,
				Email: authorEmail,
				When:  time.Now(),
			},
		}, nil
	}

	// 如果打开失败，说明不是git仓库，初始化仓库
	repo, err = git.PlainInit(path, false)
	if err != nil {
		return nil, fmt.Errorf("初始化仓库失败: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("获取工作树失败: %v", err)
	}

	return &GitProject{
		repo:     repo,
		worktree: worktree,
		path:     path,
		author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	}, nil
}

// Add 添加文件到暂存区
func (g *GitProject) Add(path string) error {
	_, err := g.worktree.Add(path)
	if err != nil {
		return fmt.Errorf("添加文件失败: %v", err)
	}
	return nil
}

// Commit 提交更改
func (g *GitProject) Commit(message string) (string, error) {
	hash, err := g.worktree.Commit(message, &git.CommitOptions{
		Author: g.author,
	})
	if err != nil {
		return "", fmt.Errorf("提交失败: %v", err)
	}
	return hash.String(), nil
}

// Reset 重置到指定的提交
func (g *GitProject) Reset(commitHash string) error {
	err := g.worktree.Reset(&git.ResetOptions{
		Mode:   git.HardReset,
		Commit: plumbing.NewHash(commitHash),
	})
	if err != nil {
		return fmt.Errorf("重置失败: %v", err)
	}
	return nil
}

// ResetTo 重置到指定的提交
func (g *GitProject) ResetTo(commitHash string, mode git.ResetMode) error {
	err := g.worktree.Reset(&git.ResetOptions{
		Mode:   mode,
		Commit: plumbing.NewHash(commitHash),
	})
	if err != nil {
		return fmt.Errorf("重置失败: %v", err)
	}
	return nil
}

// ... 在现有的Commit函数后面添加 ...

// AddAll 添加所有文件到暂存区（相当于 git add .）
func (g *GitProject) AddAll() error {
	_, err := g.worktree.Add(".")
	if err != nil {
		return fmt.Errorf("添加所有文件失败: %v", err)
	}
	return nil
}

// AddAndCommit 添加指定文件并提交（相当于 git add <path> && git commit -m "message"）
func (g *GitProject) AddAndCommit(path, message string) (string, error) {
	// 先添加文件
	if err := g.Add(path); err != nil {
		return "", err
	}

	// 再提交
	return g.Commit(message)
}

// AddAllAndCommit 添加所有文件并提交（相当于 git add . && git commit -m "message"）
func (g *GitProject) AddAllAndCommit(message string) (string, error) {
	// 先添加所有文件
	if err := g.AddAll(); err != nil {
		return "", err
	}

	// 再提交
	return g.Commit(message)
}

// CommitAll 提交所有已跟踪文件的修改（相当于 git commit -am "message"）
// 注意：这个只会提交已跟踪文件的修改，不会添加新文件
func (g *GitProject) CommitAll(message string) (string, error) {
	hash, err := g.worktree.Commit(message, &git.CommitOptions{
		Author: g.author,
		All:    true, // 这个选项相当于 git commit -a
	})
	if err != nil {
		return "", fmt.Errorf("提交所有修改失败: %v", err)
	}
	return hash.String(), nil
}

// ResetToPrevious 重置到上一个提交
func (g *GitProject) ResetToPrevious(mode git.ResetMode) error {
	head, err := g.repo.Head()
	if err != nil {
		return fmt.Errorf("获取HEAD失败: %v", err)
	}

	// 获取当前提交
	currentCommit, err := g.repo.CommitObject(head.Hash())
	if err != nil {
		return fmt.Errorf("获取当前提交失败: %v", err)
	}

	// 获取父提交
	if currentCommit.NumParents() == 0 {
		return fmt.Errorf("当前提交没有父提交")
	}

	parentCommit, err := currentCommit.Parent(0)
	if err != nil {
		return fmt.Errorf("获取父提交失败: %v", err)
	}

	return g.ResetTo(parentCommit.Hash.String(), mode)
}

// ResetToHead 重置到HEAD
func (g *GitProject) ResetToHead(mode git.ResetMode) error {
	return g.ResetTo("HEAD", mode)
}

// ResetToBranch 重置到指定分支的最新提交
func (g *GitProject) ResetToBranch(branchName string, mode git.ResetMode) error {
	ref := plumbing.NewBranchReferenceName(branchName)
	refObj, err := g.repo.Reference(ref, true)
	if err != nil {
		return fmt.Errorf("获取分支引用失败: %v", err)
	}

	return g.ResetTo(refObj.Hash().String(), mode)
}

// CreateBranch 创建新分支
func (g *GitProject) CreateBranch(branchName string) error {
	head, err := g.repo.Head()
	if err != nil {
		return fmt.Errorf("获取HEAD失败: %v", err)
	}

	ref := plumbing.NewHashReference(
		plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branchName)),
		head.Hash(),
	)

	err = g.repo.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("创建分支失败: %v", err)
	}
	return nil
}

// Checkout 切换到指定分支
func (g *GitProject) Checkout(branchName string) error {
	err := g.worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
	})
	if err != nil {
		return fmt.Errorf("切换分支失败: %v", err)
	}
	return nil
}

// GetLog 获取提交历史
func (g *GitProject) GetLog() ([]*object.Commit, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("获取HEAD失败: %v", err)
	}

	commitIter, err := g.repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("获取日志失败: %v", err)
	}

	var commits []*object.Commit
	err = commitIter.ForEach(func(c *object.Commit) error {
		commits = append(commits, c)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("遍历提交记录失败: %v", err)
	}

	return commits, nil
}

// GetCurrentBranch 获取当前分支信息
func (g *GitProject) GetCurrentBranch() (*BranchInfo, error) {
	head, err := g.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("获取HEAD失败: %v", err)
	}

	return &BranchInfo{
		Name:   head.Name().Short(),
		Hash:   head.Hash().String(),
		IsHead: true,
	}, nil
}

// ListBranches 获取所有分支列表
func (g *GitProject) ListBranches() ([]BranchInfo, error) {
	branches, err := g.repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("获取分支列表失败: %v", err)
	}

	var branchList []BranchInfo
	head, _ := g.repo.Head()
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		branchList = append(branchList, BranchInfo{
			Name:   ref.Name().Short(),
			Hash:   ref.Hash().String(),
			IsHead: ref.Name() == head.Name(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("遍历分支失败: %v", err)
	}

	return branchList, nil
}

// DeleteBranch 删除分支
func (g *GitProject) DeleteBranch(branchName string) error {
	ref := plumbing.NewBranchReferenceName(branchName)
	err := g.repo.Storer.RemoveReference(ref)
	if err != nil {
		return fmt.Errorf("删除分支失败: %v", err)
	}
	return nil
}

// GetStatus 获取文件状态
func (g *GitProject) GetStatus() ([]FileStatus, error) {
	status, err := g.worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("获取状态失败: %v", err)
	}

	var fileStatuses []FileStatus
	for path, status := range status {
		fileStatuses = append(fileStatuses, FileStatus{
			Path:     path,
			Worktree: status.Worktree,
			Staging:  status.Staging,
		})
	}

	return fileStatuses, nil
}

// GetDiff 获取文件差异
func (g *GitProject) GetDiff(path string) (string, error) {
	status, err := g.worktree.Status()
	if err != nil {
		return "", fmt.Errorf("获取状态失败: %v", err)
	}

	fileStatus, ok := status[path]
	if !ok {
		return "", fmt.Errorf("文件不存在: %s", path)
	}

	if fileStatus.Worktree == git.Unmodified {
		return "", nil
	}

	// 获取工作目录中的文件内容
	worktreeFile, err := g.worktree.Filesystem.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %v", err)
	}
	defer worktreeFile.Close()

	// 获取HEAD中的文件内容
	head, err := g.repo.Head()
	if err != nil {
		return "", fmt.Errorf("获取HEAD失败: %v", err)
	}

	commit, err := g.repo.CommitObject(head.Hash())
	if err != nil {
		return "", fmt.Errorf("获取提交对象失败: %v", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", fmt.Errorf("获取树对象失败: %v", err)
	}

	_, err = tree.File(path)
	if err != nil {
		return "", fmt.Errorf("获取文件对象失败: %v", err)
	}

	// 这里可以添加具体的差异比较逻辑
	// 由于go-git库的限制，这里只是返回一个简单的提示
	return fmt.Sprintf("文件 %s 已被修改", path), nil
}

// FindCommitByMessage 根据提交消息查找提交
func (g *GitProject) FindCommitByMessage(messagePattern string) (*CommitInfo, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("获取HEAD失败: %v", err)
	}

	commitIter, err := g.repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("获取日志失败: %v", err)
	}

	var targetCommit *object.Commit
	err = commitIter.ForEach(func(c *object.Commit) error {
		if strings.Contains(c.Message, messagePattern) {
			targetCommit = c
			return fmt.Errorf("找到目标提交") // 使用错误来中断遍历
		}
		return nil
	})
	if err != nil && err.Error() != "找到目标提交" {
		return nil, fmt.Errorf("遍历提交记录失败: %v", err)
	}

	if targetCommit == nil {
		return nil, fmt.Errorf("未找到包含消息 '%s' 的提交", messagePattern)
	}

	return &CommitInfo{
		Hash:    targetCommit.Hash.String(),
		Message: targetCommit.Message,
		Author:  targetCommit.Author.String(),
		Time:    targetCommit.Author.When,
	}, nil
}

// ResetByCommitMessage 根据提交消息重置
func (g *GitProject) ResetByCommitMessage(messagePattern string, mode git.ResetMode) error {
	commit, err := g.FindCommitByMessage(messagePattern)
	if err != nil {
		return fmt.Errorf("查找提交失败: %v", err)
	}

	return g.ResetTo(commit.Hash, mode)
}

// FindCommitsByMessage 查找所有匹配的提交
func (g *GitProject) FindCommitsByMessage(messagePattern string) ([]CommitInfo, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("获取HEAD失败: %v", err)
	}

	commitIter, err := g.repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("获取日志失败: %v", err)
	}

	var commits []CommitInfo
	err = commitIter.ForEach(func(c *object.Commit) error {
		if strings.Contains(c.Message, messagePattern) {
			commits = append(commits, CommitInfo{
				Hash:    c.Hash.String(),
				Message: c.Message,
				Author:  c.Author.String(),
				Time:    c.Author.When,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("遍历提交记录失败: %v", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("未找到包含消息 '%s' 的提交", messagePattern)
	}

	return commits, nil
}

// ResetByCommitMessageExact 根据完整的提交消息重置
func (g *GitProject) ResetByCommitMessageExact(message string, mode git.ResetMode) error {
	ref, err := g.repo.Head()
	if err != nil {
		return fmt.Errorf("获取HEAD失败: %v", err)
	}

	commitIter, err := g.repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return fmt.Errorf("获取日志失败: %v", err)
	}

	var targetCommit *object.Commit
	err = commitIter.ForEach(func(c *object.Commit) error {
		if c.Message == message {
			targetCommit = c
			return fmt.Errorf("找到目标提交") // 使用错误来中断遍历
		}
		return nil
	})
	if err != nil && err.Error() != "找到目标提交" {
		return fmt.Errorf("遍历提交记录失败: %v", err)
	}

	if targetCommit == nil {
		return fmt.Errorf("未找到消息为 '%s' 的提交", message)
	}

	return g.ResetTo(targetCommit.Hash.String(), mode)
}

// FindCommitByCompare 使用比较函数查找提交
func (g *GitProject) FindCommitByCompare(compareFunc CommitCompareFunc, resetCommitMsg string) (*CommitInfo, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("获取HEAD失败: %v", err)
	}

	commitIter, err := g.repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("获取日志失败: %v", err)
	}

	var targetCommit *object.Commit
	err = commitIter.ForEach(func(c *object.Commit) error {
		if compareFunc(c, resetCommitMsg) {
			targetCommit = c
			return fmt.Errorf("找到目标提交") // 使用错误来中断遍历
		}
		return nil
	})
	if err != nil && err.Error() != "找到目标提交" {
		return nil, fmt.Errorf("遍历提交记录失败: %v", err)
	}

	if targetCommit == nil {
		return nil, fmt.Errorf("未找到匹配的提交")
	}

	return &CommitInfo{
		Hash:    targetCommit.Hash.String(),
		Message: targetCommit.Message,
		Author:  targetCommit.Author.String(),
		Time:    targetCommit.Author.When,
	}, nil
}

// ResetByCompare 使用比较函数重置
func (g *GitProject) ResetByCompare(compareFunc CommitCompareFunc, resetCommitMsg string, mode git.ResetMode) error {
	commit, err := g.FindCommitByCompare(compareFunc, resetCommitMsg)
	if err != nil {
		return fmt.Errorf("查找提交失败: %v", err)
	}

	return g.ResetTo(commit.Hash, mode)
}

// NewJSONCommitCompare 创建JSON提交比较函数
func NewJSONCommitCompare() CommitCompareFunc {
	return func(commit *object.Commit, resetCommitMsg string) bool {
		var commitData map[string]interface{}
		err := json.Unmarshal([]byte(commit.Message), &commitData)
		if err != nil {
			return false
		}

		var resetData map[string]interface{}
		err = json.Unmarshal([]byte(resetCommitMsg), &resetData)
		if err != nil {
			return false
		}

		// 检查所有期望的字段是否匹配
		for key, expectedValue := range resetData {
			actualValue, exists := commitData[key]
			if !exists {
				return false
			}
			if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
				return false
			}
		}

		return true
	}
}

// JSONFieldMatcher 用于匹配JSON消息中的特定字段
type JSONFieldMatcher struct {
	Field string
	Value interface{}
}

// NewJSONFieldMatcher 创建一个新的JSON字段匹配器
func NewJSONFieldMatcher(field string, value interface{}) *JSONFieldMatcher {
	return &JSONFieldMatcher{
		Field: field,
		Value: value,
	}
}

// Match 实现CommitMatcher接口
func (m *JSONFieldMatcher) Match(commit *object.Commit) (bool, error) {
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(commit.Message), &msg); err != nil {
		return false, nil // 如果消息不是JSON格式，直接返回false
	}

	// 检查字段是否存在且值匹配
	if val, exists := msg[m.Field]; exists {
		return val == m.Value, nil
	}
	return false, nil
}

// FindCommitByField 根据JSON消息中的特定字段查找提交
func (g *GitProject) FindCommitByField(field string, value interface{}) (*object.Commit, error) {
	matcher := NewJSONFieldMatcher(field, value)
	return g.FindCommitByMatcher(matcher)
}

// FindCommitsByField 根据JSON消息中的特定字段查找所有匹配的提交
func (g *GitProject) FindCommitsByField(field string, value interface{}) ([]*object.Commit, error) {
	matcher := NewJSONFieldMatcher(field, value)
	return g.FindCommitsByMatcher(matcher)
}

// ResetByJSONField 根据JSON消息中的特定字段重置到指定提交
func (g *GitProject) ResetByJSONField(field string, value interface{}, mode git.ResetMode) error {
	commit, err := g.FindCommitByField(field, value)
	if err != nil {
		return fmt.Errorf("根据字段[%s=%v]查找提交失败: %v", field, value, err)
	}
	return g.ResetTo(commit.Hash.String(), mode)
}

// FindCommitByMatcher 使用匹配器查找提交
func (g *GitProject) FindCommitByMatcher(matcher CommitMatcher) (*object.Commit, error) {
	iter, err := g.repo.Log(&git.LogOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取提交历史失败: %v", err)
	}
	defer iter.Close()

	for {
		commit, err := iter.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("遍历提交历史失败: %v", err)
		}

		matched, err := matcher.Match(commit)
		if err != nil {
			return nil, fmt.Errorf("匹配提交失败: %v", err)
		}
		if matched {
			return commit, nil
		}
	}

	return nil, fmt.Errorf("未找到匹配的提交")
}

// FindCommitsByMatcher 使用匹配器查找所有匹配的提交
func (g *GitProject) FindCommitsByMatcher(matcher CommitMatcher) ([]*object.Commit, error) {
	iter, err := g.repo.Log(&git.LogOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取提交历史失败: %v", err)
	}
	defer iter.Close()

	var commits []*object.Commit
	for {
		commit, err := iter.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("遍历提交历史失败: %v", err)
		}

		matched, err := matcher.Match(commit)
		if err != nil {
			return nil, fmt.Errorf("匹配提交失败: %v", err)
		}
		if matched {
			commits = append(commits, commit)
		}
	}

	return commits, nil
}

// CreateBranchAndCheckout 创建并切换到新分支
func (g *GitProject) CreateBranchAndCheckout(branchName string) error {
	// 创建分支
	err := g.CreateBranch(branchName)
	if err != nil {
		return fmt.Errorf("创建分支失败: %v", err)
	}

	// 切换到新分支
	err = g.Checkout(branchName)
	if err != nil {
		return fmt.Errorf("切换分支失败: %v", err)
	}

	return nil
}

// CreateBranchFrom 从指定提交创建新分支
func (g *GitProject) CreateBranchFrom(branchName, commitHash string) error {
	ref := plumbing.NewHashReference(
		plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branchName)),
		plumbing.NewHash(commitHash),
	)

	err := g.repo.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("创建分支失败: %v", err)
	}
	return nil
}

// CreateBranchAndCheckoutFrom 从指定提交创建并切换到新分支
func (g *GitProject) CreateBranchAndCheckoutFrom(branchName, commitHash string) error {
	// 创建分支
	err := g.CreateBranchFrom(branchName, commitHash)
	if err != nil {
		return fmt.Errorf("创建分支失败: %v", err)
	}

	// 切换到新分支
	err = g.Checkout(branchName)
	if err != nil {
		return fmt.Errorf("切换分支失败: %v", err)
	}

	return nil
}

// CheckoutAndReset 切换到指定分支并重置
func (g *GitProject) CheckoutAndReset(branchName string, mode git.ResetMode) error {
	// 切换分支
	err := g.Checkout(branchName)
	if err != nil {
		return fmt.Errorf("切换分支失败: %v", err)
	}

	// 重置到分支最新提交
	err = g.ResetToBranch(branchName, mode)
	if err != nil {
		return fmt.Errorf("重置分支失败: %v", err)
	}

	return nil
}

// RenameBranch 重命名分支
func (g *GitProject) RenameBranch(oldName, newName string) error {
	// 获取旧分支的引用
	oldRef := plumbing.NewBranchReferenceName(oldName)
	ref, err := g.repo.Reference(oldRef, true)
	if err != nil {
		return fmt.Errorf("获取分支引用失败: %v", err)
	}

	// 创建新分支引用
	newRef := plumbing.NewHashReference(
		plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", newName)),
		ref.Hash(),
	)

	// 设置新分支引用
	err = g.repo.Storer.SetReference(newRef)
	if err != nil {
		return fmt.Errorf("创建新分支失败: %v", err)
	}

	// 删除旧分支引用
	err = g.repo.Storer.RemoveReference(oldRef)
	if err != nil {
		return fmt.Errorf("删除旧分支失败: %v", err)
	}

	return nil
}

// GetBranchCommit 获取分支的最新提交
func (g *GitProject) GetBranchCommit(branchName string) (*object.Commit, error) {
	ref := plumbing.NewBranchReferenceName(branchName)
	refObj, err := g.repo.Reference(ref, true)
	if err != nil {
		return nil, fmt.Errorf("获取分支引用失败: %v", err)
	}

	commit, err := g.repo.CommitObject(refObj.Hash())
	if err != nil {
		return nil, fmt.Errorf("获取提交对象失败: %v", err)
	}

	return commit, nil
}

// IsBranchExists 检查分支是否存在
func (g *GitProject) IsBranchExists(branchName string) (bool, error) {
	ref := plumbing.NewBranchReferenceName(branchName)
	_, err := g.repo.Reference(ref, true)
	if err == nil {
		return true, nil
	}
	if err == plumbing.ErrReferenceNotFound {
		return false, nil
	}
	return false, fmt.Errorf("检查分支是否存在失败: %v", err)
}

// GetBranchList 获取所有分支列表（包括远程分支）
func (g *GitProject) GetBranchList() ([]BranchInfo, error) {
	// 获取本地分支
	localBranches, err := g.ListBranches()
	if err != nil {
		return nil, fmt.Errorf("获取本地分支失败: %v", err)
	}

	// 获取远程分支
	remoteBranches, err := g.repo.References()
	if err != nil {
		return nil, fmt.Errorf("获取远程分支失败: %v", err)
	}

	var branches []BranchInfo
	head, _ := g.repo.Head()

	// 添加本地分支
	branches = append(branches, localBranches...)

	// 添加远程分支
	err = remoteBranches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsRemote() {
			branches = append(branches, BranchInfo{
				Name:   ref.Name().Short(),
				Hash:   ref.Hash().String(),
				IsHead: ref.Name() == head.Name(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("遍历远程分支失败: %v", err)
	}

	return branches, nil
}
