package log

import (
	"io"
	"os"
	"testing"

	api "github.com/morning-night-dream/distributed-services-with-go/api/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestSegment(t *testing.T) {
	dir, _ := os.MkdirTemp("", "segment-test")
	defer os.RemoveAll(dir)

	want := &api.Record{Value: []byte("hello world")}

	c := Config{}
	c.Segment.MaxStoreBytes = 1024
	c.Segment.MaxIndexBytes = entWidth * 3

	s, err := newSegment(dir, 16, c)
	require.NoError(t, err)
	require.Equal(t, uint64(16), s.nextOffset)
	require.False(t, s.IsMaxed())

	for i := uint64(0); i < 3; i++ {
		off, err := s.Append(want)
		require.NoError(t, err)
		require.Equal(t, 16+i, off)

		got, err := s.Read(off)
		require.NoError(t, err)
		require.Equal(t, want.Value, got.Value)
	}

	_, err = s.Append(want)
	require.Equal(t, io.EOF, err)

	require.True(t, s.IsMaxed())
	require.NoError(t, s.Close())

	p, _ := proto.Marshal(want)
	// 4個分に設定しているから、もう一個入るのでは？
	c.Segment.MaxStoreBytes = uint64(len(p)+lenWidth) * 4
	c.Segment.MaxIndexBytes = 1024
	// すでに作成済みのbaseoffset=16のストアファイル、インデックスファイルから構築される
	// 再起動時の挙動の確認？
	s, err = newSegment(dir, 16, c)
	require.NoError(t, err)
	// 3つ分入っているはずなので、一杯のまま
	require.True(t, s.IsMaxed())

	// ストアファイルとインデックスファイルの削除
	require.NoError(t, s.Remove())

	s, err = newSegment(dir, 16, c)
	require.NoError(t, err)
	require.False(t, s.IsMaxed())
	require.NoError(t, s.Close())
}
