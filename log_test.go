package log_test

import (
	"errors"
	l "github.com/p9dev/log"
	"testing"
)

var (
	log = l.GetLogger()
)

func TestGetLogger(t *testing.T) {
	l.SetLogLevel(l.Trace)
	l.App.Store("testing")
	log.T.Ln("testing log level", l.LvlStr[l.Trace])
	log.D.Ln("testing log level", l.LvlStr[l.Debug])
	log.I.Ln("testing log level", l.LvlStr[l.Info])
	log.W.Ln("testing log level", l.LvlStr[l.Warn])
	log.E.Ln("testing log level", l.LvlStr[l.Error])
	log.F.Ln("testing log level", l.LvlStr[l.Fatal])
	log.E.Chk(errors.New("dummy error as error"))
	log.I.Chk(errors.New("dummy information check"))
	log.I.Chk(nil)

}
