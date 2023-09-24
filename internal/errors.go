package internal

import (
	"fmt"

	"golang.org/x/xerrors"
)

var Errorf = fmt.Errorf

func UseFmtErrorf() {
	Errorf = fmt.Errorf
}

func UseXerrorsErrorf() {
	Errorf = xerrors.Errorf
}
