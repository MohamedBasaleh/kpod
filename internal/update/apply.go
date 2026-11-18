package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/MohamedBasaleh/kpod/internal/atomicfile"
)

func ExtractBinary(archive []byte, name string) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("read archive: %w", err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read archive: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		base := hdr.Name
		if i := bytes.LastIndexByte([]byte(base), '/'); i >= 0 {
			base = base[i+1:]
		}
		if base != name {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(tr, 64<<20))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		return data, nil
	}
	return nil, fmt.Errorf("archive does not contain %s", name)
}

func ReplaceExecutable(path string, binary []byte) error {
	info, err := os.Stat(path)
	perm := os.FileMode(0o755)
	if err == nil {
		perm = info.Mode().Perm()
		if perm == 0 {
			perm = 0o755
		}
	}
	return atomicfile.Write(path, binary, perm)
}
