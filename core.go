package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	// Expose determines if nested FileNode are accessible outside of Comment
	Expose         = ">"
	EmitsRegex     = "^\\.(\\w+)(\\`(.+)\\`)?\\s(.+)"
	EmitsFlagRegex = "(.+?):(.+)"
	FlagSplit      = ","
)

// Configuration contains all options used to establish processing of FileNode
type Configuration struct {
	Expose            bool
	Comment           *Comment
	Plugin            *[]Plugin
	RegularExpression *[]RegularExpression
}

// Plugin contains all options used to establish processing of FileNode
type Plugin struct {
	Path string `json:"path"`
}

// RegularExpression contains all options used to establish processing of FileNode
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

// CommentBlock contains all the options used to establish a comment block on Comment
type CommentBlock struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// LineNode contains all the options used to process Plugin and RegEx functions
type LineNode struct {
	CommentBlockStart bool   `json:"commentStart,omitempty"`
	CommentBlockLine  bool   `json:"commentLine,omitempty"`
	CommentBlockEnd   bool   `json:"commentEnd,omitempty"`
	CommentLine       bool   `json:"comment,omitempty"`
	Expose            bool   `json:"expose,omitempty"`
	Value             string `json:"value,omitempty"`
	Indent            int    `json:"indent,omitempty"`
	Number            int    `json:"number,omitempty"`
}

// FileNode contains the tree structure for LineNode
type FileNode struct {
	Line       *LineNode   `json:"line,omitempty"`
	Parent     *FileNode   `json:"-"`
	ParentLine int         `json:"parent,omitempty"`
	Child      []*FileNode `json:"child,omitempty"`
}

// EmitNode contains data used by Emits
type EmitNode struct {
	Keyword string      `json:"keyword,omitempty"`
	Flag    []*EmitFlag `json:"flag,omitempty"`
	Value   string      `json:"value,omitempty"`
	Data    []*EmitNode `json:"data,omitempty"`
	Line    int         `json:"-"`
}

// EmitFlag contains options used by EmitNode
type EmitFlag struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// EmitMeta contains data used to identify the source file
type EmitMeta struct {
	File      string      `json:"file"`
	Data      []*MetaData `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// MetaData contains data used to identify the source file meta data
type MetaData struct {
	Keyword string `json:"keyword,omitempty"`
	Value   string `json:"value,omitempty"`
}

// EmitFile Emits contains the standardized data structure based on EmitNode
type EmitFile struct {
	Meta *EmitMeta   `json:"meta"`
	Data []*EmitNode `json:"data"`
}

// MarshalJSON sets the ParentLine, if available, for plugin use
func (f *FileNode) MarshalJSON() ([]byte, error) {
	if f.Parent != nil {
		if f.Parent.Line != nil {
			f.ParentLine = f.Parent.Line.Number
		}
	}
	return json.Marshal(*f)
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
		if configuration.Expose && strings.HasSuffix(value, Expose) {
			data.Expose = true
			value = strings.TrimSuffix(value, Expose)
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
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
		}
	}(file)
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
	// Sanitize
	f.Sanitize()
	// Plugins
	err, pluginErr := f.Plugin(configuration.Plugin)
	if err != nil {
		return nil, fmt.Errorf("could not generate intermediate file for plugin: %v", err)
	} else if pluginErr != nil {
		pe := make([]string, len(pluginErr))
		for _, e := range pluginErr {
			pe = append(pe, e.Error())
		}
		return nil, fmt.Errorf("could not run plugins: %v", strings.Join(pe, ","))
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

// Sanitize removes all nested instances of empty LineNodes for optimized marshalling
func (f *FileNode) Sanitize() {
	for i, c := range f.Child {
		if !c.HasCommentOrExposedLine() {
			if i < len(f.Child) {
				f.Child = append(f.Child[:i], f.Child[i+1:]...)
			}
			c.FirstNode().Sanitize()
		}
		c.Sanitize()
	}
}

// HasCommentOrExposedLine returns true if FileNode satisfies IsCommentOrExposed criteria
func (f *FileNode) HasCommentOrExposedLine() bool {
	if f.Line.IsCommentOrExposed() {
		return true
	} else if len(f.Child) > 0 {
		for _, c := range f.Child {
			if c.HasCommentOrExposedLine() {
				return true
			}
		}
	}
	return false
}

// CompileRegularExpressions caches the expression compilation before use; returns all known errors
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

// FirstNode returns the first FileNode of the FileNode tree
func (f *FileNode) FirstNode() *FileNode {
	if f.Parent != nil {
		return f.Parent.FirstNode()
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
func (f *FileNode) Plugin(plugins *[]Plugin) (intermediateError error, pluginErrors []error) {
	// Generate an intermediate file for any external executable to consume
	out := fmt.Sprintf("_temp.%v.json", time.Now().Nanosecond())
	err := f.Write(out)
	if err != nil {
		return err, nil
	}
	if plugins != nil {
		for _, run := range *plugins {
			pluginError := func() error {
				cmd := exec.Command(run.Path, out)
				err := cmd.Start()
				if err != nil {
					return err
				}
				err = cmd.Wait()
				if err != nil {
					return err
				}
				jsonFile, err := os.Open(out)
				if err != nil {
					return err
				}
				defer func(jsonFile *os.File) {
					err := jsonFile.Close()
					if err != nil {
					}
				}(jsonFile)
				byteValue, err := ioutil.ReadAll(jsonFile)
				if err != nil {
					return err
				}
				if json.Unmarshal(byteValue, &f) != nil {
					return err
				}
				return nil
			}()
			if pluginError != nil {
				pluginErrors = append(pluginErrors, pluginError)
			}
		}
	}
	err = os.Remove(out)
	if err != nil {
		return err, pluginErrors
	}
	return nil, pluginErrors
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

// Write generates and saves the FileNode to disk for use by plugins
func (f *FileNode) Write(path string) error {
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// Emit returns EmitNode from FileNode
func (f *FileNode) Emit() (*EmitNode, error) {
	regexEmits, err := regexp.Compile(EmitsRegex)
	if err != nil {
		return nil, err
	}
	regexFlag, err := regexp.Compile(EmitsFlagRegex)
	if err != nil {
		return nil, err
	}
	emits, err := f.Process(regexEmits, regexFlag)
	if err != nil {
		return nil, err
	}
	return emits, nil
}

// Process returns EmitNode based on LineNode.Value
func (f *FileNode) Process(regexEmits *regexp.Regexp, regexFlag *regexp.Regexp) (*EmitNode, error) {
	e := &EmitNode{}
	if f.Line != nil {
		e.Line = f.Line.Number
		e.Value = f.Line.Value
		match := regexEmits.FindStringSubmatch(f.Line.Value)
		if len(match) > 0 {
			e.Value = match[4]
			e.Keyword = match[1]
			if len(match[3]) > 0 {
				flags := strings.Split(match[3], FlagSplit)
				if len(flags) > 0 {
					for _, flag := range flags {
						flagData := &EmitFlag{}
						flagMatch := regexFlag.FindStringSubmatch(flag)
						if len(flagMatch) > 0 {
							flagData.Name = flagMatch[1]
							flagData.Value = flagMatch[2]
						} else {
							flagData.Value = flag
						}
						e.Flag = append(e.Flag, flagData)
					}
				}
			}
		}
	}
	for _, c := range f.Child {
		n, err := c.Process(regexEmits, regexFlag)
		if err != nil {
			return nil, err
		} else {
			e.Data = append(e.Data, n)
		}
	}
	return e, nil
}

// Write generates and saves the EmitNode to disk
func (e *EmitNode) Write(inputPath string, outputPath string, meta []*MetaData) error {
	emits := &EmitFile{
		Meta: &EmitMeta{
			File:      inputPath,
			Data:      meta,
			Timestamp: time.Now().String(),
		},
		Data: e.Data,
	}
	data, err := json.Marshal(emits)
	if err != nil {
		return err
	}
	err = os.WriteFile(outputPath, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
