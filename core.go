package core

import (
	"bufio"
	"fmt"
	"os"
	"plugin"
	"regexp"
	"strings"
	"unicode"
)

// EXPOSE determines if nested FileNode are accessible outside of Comment
const EXPOSE = ">"

// Configuration contains all options used to establish processing of FileNode
type Configuration struct {
	Expose            bool
	Comment           *Comment
	Plugin            *[]Plugin
	RegularExpression *[]RegularExpression
}

// Plugin
type Plugin struct {
	Path string `json:"path"`
}

// RegularExpression
type RegularExpression struct {
	Find     string         `json:"find"`
	Replace  string         `json:"replace"`
	Compiled *regexp.Regexp `json:"-"`
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
func Line(fileNode *FileNode, value string, configuration *Configuration) *LineNode {
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
	if strings.HasPrefix(value, configuration.Comment.Block.Start) {
		data.CommentBlockStart = true
		value = strings.TrimPrefix(value, configuration.Comment.Block.Start)
	} else if strings.HasSuffix(value, configuration.Comment.Block.End) {
		data.CommentBlockEnd = true
		value = strings.TrimSuffix(value, configuration.Comment.Block.End)
	} else if strings.HasPrefix(value, configuration.Comment.Line) {
		data.CommentLine = true
		value = strings.TrimPrefix(value, configuration.Comment.Line)
		// Expose (only through comment line)
		if configuration.Expose && strings.HasSuffix(value, EXPOSE) {
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
		f.Insert(i, Line(f, data, configuration))
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("could not scan file: %v", err)
	}
	// Plugins
	err = f.Plugin(configuration.Plugin)
	if err != nil {
		return nil, fmt.Errorf("could not run plugin: %v", err)
	}
	// Regular Expressions
	if configuration.RegularExpression != nil {
		err = configuration.CompileRegularExpressions()
		if err != nil {
			return nil, err
		}
		f.RegularExpression(configuration.RegularExpression)
	}
	return f, nil
}

// CompileRegularExpressions caches the expression compiliation before use; returns all known errors
func (c *Configuration) CompileRegularExpressions() error {
	var errors []string
	r := *c.RegularExpression
	for i, e := range r {
		object, err := regexp.Compile(e.Find)
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			r[i].Compiled = object
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("could not compile regular expression: %v", strings.Join(errors, ", "))
	}
	return nil
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
	if f.Line != nil {
		if f.Line.Indent == indent {
			return f
		}
		if f.Parent != nil {
			return f.Parent.LastIndent(indent)
		}
	}
	return nil
}

// IsCommentWithinBlock returns true if FileNode satisfies CommentBlock criteria
func (f *FileNode) IsCommentWithinBlock() bool {
	return !f.LastNode().Line.IsCommentBlockEnd() && f.LastNode().Line.IsCommentBlockStart()
}

// IsExposedWithinBlock returns true if FileNode satisfies Comment and EXPOSE criteria
func (f *FileNode) IsExposedWithinBlock() bool {
	return !f.Line.IsComment() && f.LastNode().Line.IsExposed()
}

// Insert returns a FileNode based on the provided line number and LineNode
func (f *FileNode) Insert(lineNumber int, lineNode *LineNode) *FileNode {
	lastNode := f.LastNode()
	lineNode.Number = lineNumber
	if lastNode.Line == nil || lineNode.Indent == lastNode.Line.Indent {
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
func (f *FileNode) Plugin(plugins *[]Plugin) error {
	if plugins != nil {
		for _, m := range *plugins {
			fmt.Println(m.Path)
			p, err := plugin.Open(m.Path)
			if err != nil {
				return err
			}
			fn, err := p.Lookup("FileNode")
			if err != nil {
				return err
			}
			pp, err := p.Lookup("Process")
			if err != nil {
				return err
			}
			*fn.(*FileNode) = *f
			pp.(func())()
		}
	}
	return nil
}

// RegularExpression returns updated FileNode after processing RegularExpression array
func (f *FileNode) RegularExpression(r *[]RegularExpression) {
	if f.Line != nil {
		if len(f.Line.Value) > 0 {
			for _, e := range *r {
				f.Line.Value = e.Compiled.ReplaceAllString(f.Line.Value, e.Replace)
			}
		}
	}
	for _, c := range f.Child {
		c.RegularExpression(r)
	}
}

// IsCommentBlockStart returns true if LineNode satisfies CommentBlock Start criteria
func (l *LineNode) IsCommentBlockStart() bool {
	if l == nil {
		return false
	}
	return l.CommentBlockStart
}

// IsCommentBlockEnd returns true if LineNode satisfies CommentBlock End criteria
func (l *LineNode) IsCommentBlockEnd() bool {
	if l == nil {
		return false
	}
	return l.CommentBlockEnd
}

// IsComment returns true if LineNode satisfies Comment criteria
func (l *LineNode) IsComment() bool {
	if l == nil {
		return false
	}
	return l.CommentLine || l.CommentBlockStart || l.CommentBlockLine || l.CommentBlockEnd
}

// IsExposed returns true if LineNode satisfies EXPOSE criteria
func (l *LineNode) IsExposed() bool {
	if l == nil {
		return false
	}
	return l.Expose
}

// IsCommentOrExposed returns true if IsComment or IsExposed
func (l *LineNode) IsCommentOrExposed() bool {
	return l.IsComment() || l.IsExposed()
}
