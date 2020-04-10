package iso9660

import (
	"bytes"
	"io"
	"os"
)

type Item interface {
	io.Reader
	Size() int64
	Close() error

	// private
	sectors() uint32
	meta() *itemMeta
}

func NewItemReader(r io.Reader) (Item, error) {
	switch v := r.(type) {
	case Item:
		return v, nil
	case *os.File:
		return &fileHndlr{File: v}, nil
	case *bytes.Reader:
		return &readerHndlr{Reader: v}, nil
	case *bytes.Buffer:
		return &readerHndlr{Reader: bytes.NewReader(v.Bytes())}, nil
	default:
		buf := &bytes.Buffer{}
		_, err := io.Copy(buf, r)
		if err != nil {
			return nil, err
		}
		r := bytes.NewReader(buf.Bytes())
		return &readerHndlr{Reader: r}, nil
	}
}

func bufferizeItem(r io.Reader) (Item, error) {
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, r)
	if err != nil {
		return nil, err
	}
	return &bufferHndlr{d: buf.Bytes()}, nil
}

func NewItemFile(filename string) (Item, error) {
	st, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return nil, ErrIsDir
	}

	return &filepathHndlr{path: filename, st: st}, nil
}

// fileHndlr: handles an existing open file
type fileHndlr struct {
	*os.File
	m itemMeta
}

func (f *fileHndlr) Size() int64 {
	st, err := f.File.Stat()
	if err != nil {
		// stat failed
		return -1
	}

	return st.Size()
}

func (f *fileHndlr) sectors() uint32 {
	siz := f.Size()
	if siz%int64(sectorSize) == 0 {
		return uint32(siz / int64(sectorSize))
	}
	return uint32(siz/int64(sectorSize)) + 1
}

func (f *fileHndlr) Close() error {
	// not our file, so not closing it
	return nil
}

func (f *fileHndlr) meta() *itemMeta {
	return &f.m
}

// readerHndlr: handles a bytes.Reader
type readerHndlr struct {
	*bytes.Reader
	m itemMeta
}

func (b *readerHndlr) Size() int64 {
	return int64(b.Reader.Len())
}

func (b *readerHndlr) sectors() uint32 {
	siz := b.Size()
	if siz%int64(sectorSize) == 0 {
		return uint32(siz / int64(sectorSize))
	}
	return uint32(siz/int64(sectorSize)) + 1
}

func (b *readerHndlr) Close() error {
	return nil
}

func (b *readerHndlr) meta() *itemMeta {
	return &b.m
}

// bufferHndlr: handle a []byte array
type bufferHndlr struct {
	d []byte
	r *bytes.Reader
	m itemMeta
}

func (b *bufferHndlr) Size() int64 {
	return int64(len(b.d))
}

func (b *bufferHndlr) sectors() uint32 {
	siz := b.Size()
	if siz%int64(sectorSize) == 0 {
		return uint32(siz / int64(sectorSize))
	}
	return uint32(siz/int64(sectorSize)) + 1
}

func (b *bufferHndlr) Read(p []byte) (int, error) {
	if b.r == nil {
		b.r = bytes.NewReader(b.d)
	}
	return b.r.Read(p)
}

func (b *bufferHndlr) Close() error {
	if b.r != nil {
		b.r = nil
	}
	return nil
}

func (b *bufferHndlr) meta() *itemMeta {
	return &b.m
}

// filepathHandlr: handle a file by path
type filepathHndlr struct {
	path string
	st   os.FileInfo
	f    *os.File
	m    itemMeta
}

func (f *filepathHndlr) Read(p []byte) (int, error) {
	if f.f == nil {
		var err error
		f.f, err = os.Open(f.path)
		if err != nil {
			return 0, err
		}
	}
	return f.f.Read(p)
}

func (f *filepathHndlr) Size() int64 {
	return f.st.Size()
}

func (f *filepathHndlr) sectors() uint32 {
	siz := f.Size()
	if siz%int64(sectorSize) == 0 {
		return uint32(siz / int64(sectorSize))
	}
	return uint32(siz/int64(sectorSize)) + 1
}

func (f *filepathHndlr) Close() error {
	if f.f == nil {
		return nil
	}
	err := f.f.Close()
	f.f = nil
	return err
}

func (f *filepathHndlr) meta() *itemMeta {
	return &f.m
}
