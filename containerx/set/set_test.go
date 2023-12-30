package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetx_Add(t *testing.T) {
	Addvals := []int{1, 2, 3, 1}
	s := NewMapSet[int](10)
	t.Run("Add", func(t *testing.T) {
		for _, val := range Addvals {
			s.Add(val)
		}
		assert.Equal(t, s.m, map[int]struct{}{
			1: struct{}{},
			2: struct{}{},
			3: struct{}{},
		})
	})
}

func TestSetx_Delete(t *testing.T) {
	testcases := []struct {
		name    string
		delVal  int
		setSet  map[int]struct{}
		wantSet map[int]struct{}
		isExist bool
	}{
		{
			name:   "delete val ",
			delVal: 2,
			setSet: map[int]struct{}{
				2: struct{}{},
			},
			wantSet: map[int]struct{}{},
			isExist: true,
		},
		{
			name:   "deleted val not found",
			delVal: 3,
			setSet: map[int]struct{}{
				2: struct{}{},
			},
			wantSet: map[int]struct{}{
				2: struct{}{},
			},
			isExist: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewMapSet[int](10)
			s.m = tc.setSet
			s.Delete(tc.delVal)
			assert.Equal(t, tc.wantSet, s.m)
		})
	}
}

func TestSetx_IsExist(t *testing.T) {
	s := NewMapSet[int](10)
	s.Add(1)
	testcases := []struct {
		name    string
		val     int
		isExist bool
	}{
		{
			name:    "found",
			val:     1,
			isExist: true,
		},
		{
			name:    "not fonud",
			val:     2,
			isExist: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ok := s.Exist(tc.val)
			assert.Equal(t, tc.isExist, ok)
		})
	}
}

func TestSetx_Values(t *testing.T) {
	s := NewMapSet[int](10)
	testcases := []struct {
		name    string
		setSet  map[int]struct{}
		wantval map[int]struct{}
	}{
		{
			name: "found values",
			setSet: map[int]struct{}{
				1: struct{}{},
				2: struct{}{},
				3: struct{}{},
			},
			wantval: map[int]struct{}{
				1: struct{}{},
				2: struct{}{},
				3: struct{}{},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s.m = tc.setSet
			vals := s.Keys()
			ok := equal(vals, tc.wantval)
			assert.Equal(t, true, ok)
		})
	}
}

func equal(nums []int, m map[int]struct{}) bool {
	for _, num := range nums {
		_, ok := m[num]
		if !ok {
			return false
		}
		delete(m, num)
	}
	return true && len(m) == 0
}

// goos: windows
// goarch: amd64
// pkg: github.com/wkRonin/toolkit/set
// cpu: AMD Ryzen 7 1800X Eight-Core Processor
// BenchmarkSet
// BenchmarkSet/set_add
// BenchmarkSet/set_add-16                   308017              3966 ns/op
// BenchmarkSet/map_add
// BenchmarkSet/map_add-16                   309781              3812 ns/op
// BenchmarkSet/set_del
// BenchmarkSet/set_del-16                   498452              3321 ns/op
// BenchmarkSet/map_del
// BenchmarkSet/map_del-16                   369120              3118 ns/op
// BenchmarkSet/set_exist
// BenchmarkSet/set_exist-16                 521368              2545 ns/op
// BenchmarkSet/map_exist
// BenchmarkSet/map_exist-16                 434504              2346 ns/op

func BenchmarkSet(b *testing.B) {
	b.Run("set_add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			s := NewMapSet[int](100)
			b.StartTimer()
			setadd(s)
		}
	})
	b.Run("map_add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			m := make(map[int]struct{}, 100)
			b.StartTimer()
			mapadd(m)
		}
	})
	b.Run("set_del", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			s := NewMapSet[int](100)
			setadd(s)
			b.StartTimer()
			setdel(s)
		}
	})
	b.Run("map_del", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			m := make(map[int]struct{}, 100)
			mapadd(m)
			b.StartTimer()
			mapdel(m)
		}
	})
	b.Run("set_exist", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			s := NewMapSet[int](100)
			setadd(s)
			b.StartTimer()
			setGet(s)
		}
	})
	b.Run("map_exist", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			m := make(map[int]struct{}, 100)
			mapadd(m)
			b.StartTimer()
			mapGet(m)
		}
	})

}

func setadd(s Set[int]) {
	for i := 0; i < 100; i++ {
		s.Add(i)
	}
}

func mapadd(m map[int]struct{}) {
	for i := 0; i < 100; i++ {
		m[i] = struct{}{}
	}
}

func setdel(s Set[int]) {
	for i := 0; i < 100; i++ {
		s.Delete(i)
	}
}

func mapdel(m map[int]struct{}) {
	for i := 0; i < 100; i++ {
		delete(m, i)
	}
}
func setGet(s Set[int]) {
	for i := 0; i < 100; i++ {
		_ = s.Exist(i)
	}
}

func mapGet(s map[int]struct{}) {
	for i := 0; i < 100; i++ {
		_ = s[i]
	}
}
