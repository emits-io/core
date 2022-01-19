package core

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
)

// EXPOSE determines if nested FileNode are accessible outside of Comment
const EXPOSE = ">"

// Configuration contains all options used to establish processing of FileNode
type Configuration struct {
	Comment           *Comment
	Plugin            *[]Plugin
	RegularExpression *[]RegularExpression
}

// Plugin
type Plugin struct {
	// TODO:Plugin Fields
}

// RegularExpression
type RegularExpression struct {
	// TODO: Regular Expressions Fields
}

// Comment contains all the options used to establish a comment on LineNode
type Comment struct {
	Line  string        `json:"line"`
	Block *CommentBlock `json:"block"`
}

// CommentBlock contains all of the options used to establish a comment block on Comment
type CommentBlock struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// LineNode contains all of the options used to process Plugin and RegEx functions
type LineNode struct {
	CommentBlockStart bool
	CommentBlockLine  bool
	CommentBlockEnd   bool
	CommentLine       bool
	Expose            bool
	Value             string
	Indent            int
	Number            int
}

// FileNode contains the tree structure for LineNode
type FileNode struct {
	Line   *LineNode
	Parent *FileNode
	Child  []*FileNode
}

// Line returns LineNode
func Line(fileNode *FileNode, value string, comment *Comment) *LineNode {
	// Indent
	indent := 0
	for i, r := range value {
		indent = i
		if !unicode.IsSpace(r) {
			break
		}
	}
	data := &LineNode{
		Indent: indent,
	}
	value = value[indent:]
	// Explicit Comment
	if strings.HasPrefix(value, comment.Block.Start) {
		data.CommentBlockStart = true
		value = strings.TrimPrefix(value, comment.Block.Start)
	} else if strings.HasSuffix(value, comment.Block.End) {
		data.CommentBlockEnd = true
		value = strings.TrimSuffix(value, comment.Block.End)
	} else if strings.HasPrefix(value, comment.Line) {
		data.CommentLine = true
		value = strings.TrimPrefix(value, comment.Line)
		// Expose (only through comment line)
		if strings.HasSuffix(value, EXPOSE) {
			data.Expose = true
			value = strings.TrimSuffix(value, EXPOSE)
		}
	} else {
		// Possible Comment
		data.CommentBlockLine = fileNode.IsCommentWithinBlock()
		// Possible Expose
		data.Expose = fileNode.IsExposedWithinBlock()
	}
	// Possible Value
	if data.IsCommentOrExposed() {
		data.Value = strings.TrimSpace(value)
	}
	return data
}

// Build opens the provided file path and returns a FileNode based on Configuration
func (f *FileNode) Build(path string, configuration *Configuration) (*FileNode, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()
	sc := bufio.NewScanner(file)
	i := 0
	for sc.Scan() {
		i++
		data := sc.Text()
		f.Insert(i, Line(f, data, configuration.Comment))
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("could not scan file: %v", err)
	}
	err = f.Plugin(configuration.Plugin)
	if err != nil {
		return nil, fmt.Errorf("could not run plugin: %v", err)
	}
	err = f.RegularExpression(configuration.RegularExpression)
	if err != nil {
		return nil, fmt.Errorf("could not perform regular expression: %v", err)
	}
	return f, nil
}

// LastNode returns the last FileNode of the last FileNode.Child
func (f *FileNode) LastNode() *FileNode {
	if f.Child != nil {
		return f.Child[len(f.Child)-1].LastNode()
	}
	return f
}

// LastIndent returns the last FileNode with the provided indent, or the last FileNode if not found
func (f *FileNode) LastIndent(indent int) *FileNode {
	if f.Line.Indent == indent {
		return f
	}
	if f.Parent != nil {
		return f.Parent.LastIndent(indent)
	}
	return nil
}

// IsCommentWithinBlock returns true if FileNode satisfies CommentBlock criteria
func (f *FileNode) IsCommentWithinBlock() bool {
	if !f.LastNode().Line.CommentBlockEnd {
		if f.LastNode().Line.CommentBlockStart || f.LastNode().Line.CommentBlockLine {
			return true
		}
	}
	return false
}

// IsExposedWithinBlock returns true if FileNode satisfies EXPOSE criteria
func (f *FileNode) IsExposedWithinBlock() bool {
	if !f.Line.IsComment() && f.LastNode().Line.Expose {
		return true
	}
	return false
}

// Insert returns a FileNode based on the provided line number and LineNode
func (f *FileNode) Insert(lineNumber int, lineNode *LineNode) *FileNode {
	lastNode := f.LastNode()
	lineNode.Number = lineNumber
	if lineNode.Indent == lastNode.Line.Indent {
		if lastNode.Parent != nil {
			lastNode.Parent.Child = append(lastNode.Parent.Child, &FileNode{
				Line:   lineNode,
				Parent: lastNode.Parent,
			})
		} else {
			lastNode.Child = append(lastNode.Child, &FileNode{
				Line:   lineNode,
				Parent: lastNode,
			})
		}
	} else if lineNode.Indent > lastNode.Line.Indent {
		lastNode.Child = append(lastNode.Child, &FileNode{
			Line:   lineNode,
			Parent: lastNode,
		})
	} else if lineNode.Indent < lastNode.Line.Indent {
		lastIndent := lastNode.LastIndent(lineNode.Indent)
		if lastIndent != nil {
			lastIndent.Parent.Child = append(lastIndent.Parent.Child, &FileNode{
				Line:   lineNode,
				Parent: lastIndent.Parent,
			})
		} else {
			lastNode.Child = append(lastNode.Child, &FileNode{
				Line:   lineNode,
				Parent: lastNode,
			})
		}
	}
	return f
}

// Plugin returns updated FileNode after processing Plugin array
func (f *FileNode) Plugin(p *[]Plugin) error {
	// TODO: Plugin Logic
	return nil
}

// RegularExpression returns updated FileNode after processing RegularExpression array
func (f *FileNode) RegularExpression(r *[]RegularExpression) error {
	if len(f.Line.Value) > 0 {
		// TODO: Regular Expressions Logic
	}
	for _, c := range f.Child {
		c.RegularExpression(r)
	}
	return nil
}

// IsComment returns true if LineNode satisfies Comment criteria
func (l *LineNode) IsComment() bool {
	return l.CommentLine || l.CommentBlockStart || l.CommentBlockLine || l.CommentBlockEnd
}

// IsExposed returns true if LineNode satisfies EXPOSE criteria
func (l *LineNode) IsExposed() bool {
	return l.Expose
}

// IsCommentOrExposed returns true if IsComment or IsExposed
func (l *LineNode) IsCommentOrExposed() bool {
	return l.IsComment() || l.IsExposed()
}