package Test

import "pkg.deepin.io/lib/dbusutil"

type testStruct struct {
	tInterface TestInterface
}

func (t1 *testStruct) Test2() {
	t1.tInterface = TestInterface{}
	testService := &dbusutil.Service{}
	testService.Export(testpath, &t1.tInterface)
}

type TestInterface struct {
}

const (
	testpath = "Path is test"
)

func (t *TestInterface) GetInterfaceName() string {
	return testpath
}
