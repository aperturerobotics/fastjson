package fastjson

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// Generic test driver that tests both serial and concurrent executions of the test with a given arena
func testArenaDriver(t *testing.T, iterations int, doTest func(a *Arena) error) {
	t.Helper()
	t.Run("serial", func(t *testing.T) {
		var a Arena
		for i := 0; i < iterations; i++ {
			if err := doTest(&a); err != nil {
				t.Fatal(err)
			}
			a.Reset()
		}
	})
	t.Run("concurrent", func(t *testing.T) {
		var ap ArenaPool
		workers := 128
		ch := make(chan error, workers)
		for i := 0; i < workers; i++ {
			go func() {
				a := ap.Get()
				defer func() {
					a.Reset()
					ap.Put(a)
				}()
				var err error
				for i := 0; i < iterations; i++ {
					if err = doTest(a); err != nil {
						break
					}
				}
				ch <- err
			}()
		}
		for i := 0; i < workers; i++ {
			select {
			case err := <-ch:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second * 3):
				t.Fatalf("timeout")
			}
		}
	})
}

func TestArena(t *testing.T) {
	testArenaDriver(t, 1000, func(a *Arena) error { return testArena(a) })
}

func testArena(a *Arena) error {
	o := a.NewObject()
	o.Set("nil1", a.NewNull())
	o.Set("nil2", nil)
	o.Set("false", a.NewFalse())
	o.Set("true", a.NewTrue())
	ni := a.NewNumberInt(123)
	o.Set("ni", ni)
	o.Set("nf", a.NewNumberFloat64(1.23))
	o.Set("ns", a.NewNumberString("34.43"))
	o.Set("nbs", a.NewNumberStringBytes(s2b("-98.765")))
	s := a.NewString("foo")
	o.Set("str1", s)
	o.Set("str2", a.NewStringBytes([]byte("xx")))

	aa := a.NewArray()
	aa.SetArrayItem(0, s)
	aa.Set("1", ni)
	o.Set("a", aa)
	obj := a.NewObject()
	obj.Set("s", s)
	o.Set("obj", obj)

	str := o.String()
	strExpected := `{"nil1":null,"nil2":null,"false":false,"true":true,"ni":123,"nf":1.23,"ns":34.43,"nbs":-98.765,"str1":"foo","str2":"xx","a":["foo",123],"obj":{"s":"foo"}}`
	if str != strExpected {
		return fmt.Errorf("unexpected json\ngot\n%s\nwant\n%s", str, strExpected)
	}
	return nil
}

func TestArenaDeepCopyValue(t *testing.T) {
	testArenaDriver(t, 100, func(a *Arena) error { return testArenaDeepCopyValue(a) })
}

func randValidChar() rune {
	for {
		c := rune('0' + rand.Intn(78))
		if c != '\\' {
			return c
		}
	}
}

func testArenaDeepCopyValue(a *Arena) error {
	const jsonTest = `{"nil":null,"false":false,"true":true,"ni":123,"nf":1.23,"ns":34.43,"nbs":-98.765,"str1":"foo","str2":"xx","a":["foo",123,{"s":"x","n":-1.0,"o":{},"nil":null}],"obj":{"s":"foo","a":[123,"s",{"f":{"f2":"v","f3":{"a":98}}}]}}`
	// Use a locally controlled parser that we'll reset and reuse to ensure the deep copy truly copied all the values
	var p Parser
	tempValue, err := p.Parse(jsonTest)
	if err != nil {
		return fmt.Errorf("failed to parse test json: %w", err)
	}
	// Validate that the serialized value matches the original test
	tempSerialized := b2s(tempValue.MarshalTo(nil))
	if tempSerialized != jsonTest {
		return fmt.Errorf("initial parsed test JSON does not match\ngot\n%s\nwant\n%s", tempSerialized, jsonTest)
	}
	// Do a deep copy to preserve the values after the parser is reused
	shallowCopy := tempValue
	deepCopy := a.DeepCopyValue(tempValue)
	// Now reuse the parser enough times so it should trash a shallow copy
	for i := 0; i < 100; i++ {
		rn := rand.Int63n(2 ^ 40)
		rs := fmt.Sprintf("%c%d%d%c", randValidChar(), rand.Int63(), rand.Int63(), randValidChar())
		mixerJSON := fmt.Sprintf(`{"n1":%d,"s1":"%s","o1":{"a1":[%d,true,%d,false,"%s",{"f1":%d,"f2":[%d,%d,[%d,"%s"]]}]}}`, rn, rs, rn, rn, rs, rn, rn, rn, rn, rs)
		_, err = p.Parse(mixerJSON)
		if err != nil {
			return fmt.Errorf("failed reusing parser to parser random JSON: %w\nJSON\n%s", err, mixerJSON)
		}
	}
	// Now check that the deep copy is good and the shallow copy is bad
	deepCopyJSON := b2s(deepCopy.MarshalTo(nil))
	if deepCopyJSON != jsonTest {
		return fmt.Errorf("deep copy JSON does not match\ngot\n%s\nwant\n%s", deepCopyJSON, jsonTest)
	}
	shallowCopyJSON := b2s(shallowCopy.MarshalTo(nil))
	if shallowCopyJSON == jsonTest {
		return fmt.Errorf("shallow copy JSON matches when it should not match!\nshallow_copy\n%s", shallowCopyJSON)
	}
	return nil
}
