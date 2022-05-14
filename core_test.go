package core_test

import (
	"regexp"
	"testing"

	"github.com/emits-io/core"
)

func Test_Build(t *testing.T) {
	r := make([]core.RegularExpression, 0)
	r = append(r, core.RegularExpression{
		Find:    "test",
		Replace: "bar",
	})
	f := &core.FileNode{}
	_, err := f.Build("core.go", &core.Configuration{
		Comment: &core.Comment{
			Line: "//",
			Block: &core.CommentBlock{
				Start: "/*",
				End:   "*/",
			},
		},
		Plugin: &[]core.Plugin{
			{
				"./foo.js",
			},
			{
				"./bar.js",
			},
		},
		RegularExpression: &r,
	})
	if err != nil {
		t.Errorf("Build() expects nil, got %s", err)
	}
	emits, err := f.Emit()
	if err != nil {
		t.Errorf("Emit() expects nil, got %s", err)
	}
	var m = make([]*core.MetaData, 0)
	m = append(m, &core.MetaData{
		Keyword: "layout",
		Value:   "foo",
	})
	err = emits.Write("core.go", "test.txt.json", m)
	if err != nil {
		t.Errorf("Write() expects nil, got %s", err)
	}
	if err != nil {
		t.Errorf("Build() expects nil, got %s", err)
	}
}

func Test_Build_Error(t *testing.T) {
	f := &core.FileNode{}
	_, err := f.Build("", &core.Configuration{})
	if err == nil {
		t.Errorf("Build() expects error, got %v", err)
	}
}

func Test_Build_RegularExpression_Error(t *testing.T) {
	r := make([]core.RegularExpression, 0)
	r = append(r, core.RegularExpression{
		Find: "a(",
	})
	f := &core.FileNode{}
	_, err := f.Build("core.go", &core.Configuration{
		Comment: &core.Comment{
			Line: "//",
			Block: &core.CommentBlock{
				Start: "/*",
				End:   "*/",
			},
		},
		RegularExpression: &r,
	})
	if err == nil {
		t.Errorf("Build() expects error, got %v", err)
	}
}

func Test_Line_IsComment(t *testing.T) {
	l := core.Line(&core.FileNode{}, "//", &core.Configuration{
		Comment: &core.Comment{
			Line: "//",
			Block: &core.CommentBlock{
				Start: "/*",
				End:   "*/",
			},
		},
	})
	b := l.IsComment()
	if !b {
		t.Errorf("IsComment() expects true, got %v", b)
	}
}
func Test_Line_IsCommentBlockStart(t *testing.T) {
	l := core.Line(&core.FileNode{}, "/*", &core.Configuration{
		Comment: &core.Comment{
			Line: "//",
			Block: &core.CommentBlock{
				Start: "/*",
				End:   "*/",
			},
		},
	})
	b := l.IsCommentBlockStart()
	if !b {
		t.Errorf("IsCommentBlockStart() expecting true, got %v", b)
	}
}

func Test_Line_IsCommentBlockEnd(t *testing.T) {
	l := core.Line(&core.FileNode{}, "*/", &core.Configuration{
		Comment: &core.Comment{
			Line: "//",
			Block: &core.CommentBlock{
				Start: "/*",
				End:   "*/",
			},
		},
	})
	b := l.IsCommentBlockEnd()
	if !b {
		t.Errorf("IsCommentBlockEnd() expects true, got %v", b)
	}
}

func Test_Line_IsExposed(t *testing.T) {
	l := core.Line(&core.FileNode{}, "// >", &core.Configuration{
		Expose: true,
		Comment: &core.Comment{
			Line: "//",
			Block: &core.CommentBlock{
				Start: "/*",
				End:   "*/",
			},
		},
	})
	b := l.IsExposed()
	if !b {
		t.Errorf("IsExposed() expects true, got %v", b)
	}
}

func Test_Line_IsExposed_False(t *testing.T) {
	l := core.LineNode{}
	b := l.IsExposed()
	if b {
		t.Errorf("IsExposed() expects false, got %v", b)
	}
}

func Test_Line_Insert(t *testing.T) {
	f := &core.FileNode{}
	f.Insert(1, &core.LineNode{
		Indent: 4,
	})
	f.Insert(2, &core.LineNode{
		Indent: 2,
	})
	if f.Child[0].Line.Indent != 4 {
		t.Errorf("Insert() line indent expects 4, got %v", f.Child[0].Line.Indent)
	}
	if f.Child[0].Child[0].Line.Indent != 2 {
		t.Errorf("Insert() line indent expects 2, got %v", f.Child[0].Child[0].Line.Indent)
	}
}

func Test_File_LastIndent(t *testing.T) {
	c := make([]*core.FileNode, 0)
	c = append(c, &core.FileNode{
		Line: &core.LineNode{
			Indent: 3,
		},
	})
	f := &core.FileNode{
		Line: &core.LineNode{
			Indent: 5,
		},
		Child: c,
	}
	n := f.LastNode().LastIndent(3)
	if n == nil || n.Line.Indent != 3 {
		t.Errorf("LastIndent(3) expects 3, got %v", n.Line.Indent)
	}
}

func Test_File_LastIndent_Nil(t *testing.T) {
	c := make([]*core.FileNode, 0)
	c = append(c, &core.FileNode{
		Line: &core.LineNode{
			Indent: 3,
		},
	})
	f := &core.FileNode{
		Line: &core.LineNode{
			Indent: 5,
		},
		Child: c,
	}
	n := f.LastNode().LastIndent(1)
	if n != nil {
		t.Errorf("LastIndent(1) expects nil, got %v", n.Line.Indent)
	}
}

func Test_CompileRegularExpressions(t *testing.T) {
	r := make([]core.RegularExpression, 0)
	r = append(r, core.RegularExpression{
		Find: "a(-b]+c)",
	})
	configuration := core.Configuration{
		RegularExpression: &r,
	}
	err := configuration.CompileRegularExpressions()
	if err != nil {
		t.Errorf("CompileRegularExpressions() expects nil, got %v", err)
	}
}

func Test_CompileRegularExpressions_Error(t *testing.T) {
	r := make([]core.RegularExpression, 0)
	r = append(r, core.RegularExpression{
		Find: "a(-b]+c",
	})
	configuration := core.Configuration{
		RegularExpression: &r,
	}
	err := configuration.CompileRegularExpressions()
	if err == nil {
		t.Errorf("CompileRegularExpressions() expects error, got %v", err)
	}
}

func Test_Process_RegularExpression_Flag_Array(t *testing.T) {
	regexEmits, err := regexp.Compile(core.EmitsRegex)
	if err != nil {
		t.Errorf("Process() expects nil, got %v", err)
	}
	regexFlag, err := regexp.Compile(core.EmitsFlagRegex)
	if err != nil {
		t.Errorf("Process() expects nil, got %v", err)
	}
	n := core.FileNode{
		Line: &core.LineNode{
			Value: ".keyword`flag:flag_value,foo:world` value",
		},
	}
	_, err = n.Process(regexEmits, regexFlag)
	if err != nil {
		t.Errorf("Process() expects nil, got %v", err)
	}
}

func Test_Process_RegularExpression_Flag_String(t *testing.T) {
	regexEmits, err := regexp.Compile(core.EmitsRegex)
	if err != nil {
		t.Errorf("Process() expects nil, got %v", err)
	}
	regexFlag, err := regexp.Compile(core.EmitsFlagRegex)
	if err != nil {
		t.Errorf("Process() expects nil, got %v", err)
	}
	n := core.FileNode{
		Line: &core.LineNode{
			Value: ".keyword`hello world` value",
		},
	}
	_, err = n.Process(regexEmits, regexFlag)
	if err != nil {
		t.Errorf("Process() expects nil, got %v", err)
	}
}

func Test_File_Write_Error(t *testing.T){
	n := core.EmitNode{}
	err := n.Write("/null","/null", nil)
	if err == nil {
		t.Errorf("Write() expects error, got nil")
	}
}