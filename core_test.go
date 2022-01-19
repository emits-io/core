package core_test

import (
	"github.com/emits-io/core/v1"
	"testing"
)

func TestLineNode_CommentBlockStartIsComment(t *testing.T) {
	lineNode := core.LineNode{
		CommentBlockStart: true,
	}
	value := lineNode.IsComment()
	if value != true {
		t.Errorf("IsComment() = %v; want true", value)
	}
}

func TestLineNode_CommentBlockLineIsComment(t *testing.T) {
	lineNode := core.LineNode{
		CommentBlockLine: true,
	}
	value := lineNode.IsComment()
	if value != true {
		t.Errorf("IsComment() = %v; want true", value)
	}
}

func TestLineNode_CommentBlockEndIsComment(t *testing.T) {
	lineNode := core.LineNode{
		CommentBlockEnd: true,
	}
	value := lineNode.IsComment()
	if value != true {
		t.Errorf("IsComment() = %v; want true", value)
	}
}

func TestLineNode_CommentLineIsComment(t *testing.T) {
	lineNode := core.LineNode{
		CommentBlockLine: true,
	}
	value := lineNode.IsComment()
	if value != true {
		t.Errorf("IsComment() = %v; want true", value)
	}
}
