package pocket

type Mype interface {
	IsPlural() bool
	IsSingle() bool
	IsSingleType(int) bool
	GetSingleType() int
	IsEmpty() bool
	IsFull() bool
	Intersection(Mype) Mype
	Union(Mype) Mype
	WouldChangeFromUnionWith(Mype) bool
	WouldChangeFromIntersectionWith(Mype) bool
	Subtract(Mype) Mype
	Converse() Mype
	ContainsSingleType(int) bool
	ContainsAnyType([]int) bool
	ToType() int // only works if IsSingle() is true
}

type MypeExplicit struct {
	Types map[int]bool
}

var _ Mype = &MypeExplicit{} // interface implementation declaration

const (
	MypeExplicitFullCount = 7 // must correspond to the len of types returned by MypeExplicitNewFull()
)

func MypeExplicitNewFull() *MypeExplicit {
	return &MypeExplicit{
		Types: map[int]bool{
			TY_INT:    true,
			TY_STRING: true,
			TY_FLOAT:  true,
			TY_BOOL:   true,
			TY_LIST:   true,
			TY_MAP:    true,
			TY_SET:    true,
		},
	}
}

func MypeExplicitNewSingle(singleType int) *MypeExplicit {
	return &MypeExplicit{
		Types: map[int]bool{
			singleType: true,
		},
	}
}

func MypeExplicitNewEmpty() *MypeExplicit {
	return &MypeExplicit{
		Types: map[int]bool{},
	}
}

func (me *MypeExplicit) IsPlural() bool {
	return len(me.Types) > 1
}

func (me *MypeExplicit) IsSingle() bool {
	return len(me.Types) == 1
}

func (me *MypeExplicit) IsFull() bool {
	return len(me.Types) == MypeExplicitFullCount
}

func (me *MypeExplicit) IsSingleType(t int) bool {
	if me.IsSingle() {
		if _, ok := me.Types[t]; ok {
			return true
		}
	}
	return false
}

func (me *MypeExplicit) ContainsSingleType(t int) bool {
	if _, ok := me.Types[t]; ok {
		return true
	}
	return false
}

func (me *MypeExplicit) ContainsAnyType(ts []int) bool {
	for _, t := range ts {
		if _, ok := me.Types[t]; ok {
			return true
		}
	}
	return false
}

func (me *MypeExplicit) GetSingleType() int {
	for key := range me.Types {
		return key
	}
	panic("no types in mype")
}

func (me *MypeExplicit) IsEmpty() bool {
	return len(me.Types) == 0
}

func (me *MypeExplicit) Intersection(m Mype) Mype {
	if otherMe, ok := m.(*MypeExplicit); ok {
		outTypes := make(map[int]bool)
		for myKey := range me.Types {
			if _, ok := otherMe.Types[myKey]; ok {
				outTypes[myKey] = true
			}
		}
		for otherKey := range otherMe.Types {
			if _, ok := me.Types[otherKey]; ok {
				outTypes[otherKey] = true
			}
		}
		return &MypeExplicit{
			Types: outTypes,
		}
	} else {
		panic("must intersect with mypeexplicit")
	}
}

func (me *MypeExplicit) Union(m Mype) Mype {
	if otherMe, ok := m.(*MypeExplicit); ok {
		outTypes := make(map[int]bool)
		for myKey := range me.Types {
			outTypes[myKey] = true
		}
		for otherKey := range otherMe.Types {
			outTypes[otherKey] = true
		}
		return &MypeExplicit{
			Types: outTypes,
		}
	} else {
		panic("must intersect with mypeexplicit")
	}
}

func (me *MypeExplicit) Converse() Mype {
	return MypeExplicitNewFull().Subtract(me)
}

func (me *MypeExplicit) Subtract(other Mype) Mype {
	if otherMe, ok := other.(*MypeExplicit); ok {
		outTypes := make(map[int]bool)
		for myKey := range me.Types {
			outTypes[myKey] = true
		}
		for otherKey := range otherMe.Types {
			if _, ok := outTypes[otherKey]; ok { // if other key existed in us, remove it from us
				delete(outTypes, otherKey)
			}
		}
		return &MypeExplicit{
			Types: outTypes,
		}
	} else {
		panic("must subtract with mypeexplicit")
	}
}

func (me *MypeExplicit) WouldChangeFromUnionWith(other Mype) bool {
	if otherMe, ok := other.(*MypeExplicit); ok {
		for otherType := range otherMe.Types {
			if _, ok := me.Types[otherType]; !ok { // if a type was found in the other that wasn't in us
				return true
			}
		}
		return false
	} else {
		panic("unsupported mype arg type")
	}
}

func (me *MypeExplicit) WouldChangeFromIntersectionWith(other Mype) bool {
	if otherMe, ok := other.(*MypeExplicit); ok {
		allMineInOther := true
		for myType := range me.Types {
			if _, ok := otherMe.Types[myType]; !ok { // if a type was found in us that wasn't in other
				allMineInOther = false
				break
			}
		}
		return !allMineInOther
	} else {
		panic("unsupported mype arg type")
	}
}

func (me *MypeExplicit) ToType() int {
	panic("not implemented")
}
