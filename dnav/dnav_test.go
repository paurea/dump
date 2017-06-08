package dnav_test

import (
	"dump/dnav"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestSameDate(t *testing.T) {
	d1 := dnav.NewDumpDate(2007, 3, 23, 0)
	d1Eq := dnav.NewDumpDate(2007, 3, 23, 0)
	d2 := dnav.NewDumpDate(2008, 4, 25, 1)

	if !d1.SameYear(d1Eq) || !d1.SameMonth(d1Eq) || !d1.SameDay(d1Eq) || !d1.SameHour(d1Eq) {
		t.Fatalf("should be the same")
	}
	if d2.SameYear(d1Eq) || d2.SameMonth(d1Eq) || d2.SameDay(d1Eq) || d2.SameHour(d1Eq) {
		t.Fatalf("should not be the same")
	}
}

const HoursInYear = 8760 //approx

func TestSumDates(t *testing.T) {
	d1 := dnav.NewDumpDate(2007, 3, 23, 0)
	d2 := dnav.NewDumpDate(0, 0, 0, 1)

	dSum := d1
	for i := 0; i < 2*HoursInYear; i++ {
		dSum := dnav.SumDates(*dSum, *d2)
		if !dSum.IsAfter(*d1) {
			t.Fatalf("sum does not increment")
		}
	}
}

func TestConvDates(t *testing.T) {
	d := dnav.NewDumpDate(2007, 3, 23, 0)
	zD := dnav.NewDumpDate(1, 1, 1, 0) //Zero date in go
	var ts time.Time

	dSum := dnav.TimeAddDate(ts, *d)
	d2sum := dnav.SumDates(*d, *zD)
	if !dSum.SameYear(&d2sum) || !dSum.SameMonth(&d2sum) || !dSum.SameDay(&d2sum) || !dSum.SameHour(&d2sum) {
		t.Fatalf("should be the same %s+%s= %s", ts, d, &dSum)
	}
}

func TestRoots(t *testing.T) {
	var r dnav.Roots
	os.Setenv(dnav.MainRootVar, "/adfadf:/2rsdfewr2/asf3qer:/bin")
	os.Setenv(dnav.MainDumpVar, "/adf13123adf:/sdfsd/asf3qer:/etc")
	dnav.RdRoots(&r)
	if r.MainRoot != "/bin" || r.DumpRoot != "/etc" || r.RootName != "bin" {
		t.Fatalf("bad roots %s", r)
	}
}

const (
	GoodDumpPath    = "/etc/2017/0415/0036/bin"
	BadDumpPath     = "/etc/2017/patatilla"
	BadDumpPathRoot = "/blabla/2017/0415/0036"
)

func TestParseDump(t *testing.T) {
	var r dnav.Roots
	os.Setenv(dnav.MainRootVar, "/adfadf:/2rsdfewr2/asf3qer:/bin")
	os.Setenv(dnav.MainDumpVar, "/adf13123adf:/sdfsd/asf3qer:/etc")
	dnav.RdRoots(&r)

	d, err := dnav.ParseDumpPath(GoodDumpPath, r)
	if err != nil {
		t.Fatalf("should not error, good path %s: %s", GoodDumpPath, err)
	}
	dShould := *dnav.NewDumpDate(2017, 4, 15, 36)
	if dShould != d {
		t.Fatalf("should be equal %s and %s", d, dShould)
	}
	d, err = dnav.ParseDumpPath(BadDumpPath, r)
	if err == nil {
		t.Fatalf("should error, bad path %s: %s", BadDumpPath, err)
	}
	d, err = dnav.ParseDumpPath(BadDumpPathRoot, r)
	if err == nil {
		t.Fatalf("should error, bad dump root [%s] path %s: %s", r.DumpRoot, BadDumpPathRoot, err)
	}
}

const (
	TmpDumpRootBase = "/tmp/e"
	ShouldBePath    = "/2003/0810/1030"
)

func TestFindDump(t *testing.T) {
	var r dnav.Roots

	tmproot := TmpDumpRootBase + "shouldnotexist"
	os.RemoveAll(tmproot)
	os.Setenv(dnav.MainRootVar, "/adfadf:/2rsdfewr2/asf3qer:/bin")
	os.Mkdir(tmproot, 0700)
	os.Setenv(dnav.MainDumpVar, "/adf13123adf:/sdfsd/asf3qer:"+tmproot)
	dnav.RdRoots(&r)

	for y := 2000; y < 2005; y++ {
		for m := 8; m < 11; m++ {
			for d := 5; d < 12; d++ {
				for h := 900; h < 1100; h += 100 {
					for min := 0; min < 45; min += 15 {
						p := fmt.Sprintf("%s/%4.4d/%2.2d%2.2d/%4.4d/bin", tmproot, y, d, m, h+min)
						os.MkdirAll(p, 0700)
					}
				}
			}
		}
	}
	d := dnav.NewDumpDate(2003, 8, 10, 1100)
	p := dnav.FindDumpPath(*d, r)
	if p != tmproot+ShouldBePath {
		t.Fatalf("should be equal %s %s\n", p, tmproot+ShouldBePath)
	}
	os.RemoveAll(tmproot)
}
