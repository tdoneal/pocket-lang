package pocket

type Mype interface {
	IsSingle() bool
	IsEmpty() bool
	Intersection(Mype) Mype
	ToType() int // only works if IsSingle() is true
}

type MypeExplicit struct {
	Types map[int]bool
}

var _ Mype = &MypeExplicit{} // interface implementation declaration

func MypeExplicitNewFull() *MypeExplicit {
	return &MypeExplicit{
		Types: map[int]bool{
			TY_INT:    true,
			TY_STRING: true,
			TY_FLOAT:  true,
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

func (me *MypeExplicit) IsSingle() bool {
	return len(me.Types) == 1
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

func (me *MypeExplicit) ToType() int {
	panic("not implemented")
}
