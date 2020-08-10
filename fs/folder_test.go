package fs

import (
	"testing"
)

func TestNode_nextHierarchyLevel(t *testing.T) {
	type args struct {
		path   string
		parent string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{
			name:  "root",
			args:  args{path: "/foo", parent: "/"},
			want:  "foo",
			want1: false,
		}, {
			name:  "root - has more",
			args:  args{path: "/foo/bar", parent: "/"},
			want:  "foo",
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{}
			level, hasMore := n.nextHierarchyLevel(tt.args.path, tt.args.parent)
			if level != tt.want {
				t.Errorf("nextHierarchyLevel() level = %v, want %v", level, tt.want)
			}
			if hasMore != tt.want1 {
				t.Errorf("nextHierarchyLevel() hasMore = %v, want %v", hasMore, tt.want1)
			}
		})
	}
}

func TestNode_absPath(t *testing.T) {
	type args struct {
		parent   string
		fileName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal case",
			args: args{"", "aaa"},
			want: "/aaa",
		}, {
			name: "no file name",
			args: args{"", ""},
			want: "/",
		}, {
			name: "under parent",
			args: args{"/aweme", ""},
			want: "/aweme/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{path: tt.args.parent}
			if got := n.resolve(tt.args.fileName); got != tt.want {
				t.Errorf("resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}
