package schottky

// Order combines value direction with null placement.
type Order uint8

const (
	AscNullsFirst Order = iota
	AscNullsLast
	DescNullsFirst
	DescNullsLast
)

func (o Order) valid() bool {
	return o <= DescNullsLast
}

func (o Order) descending() bool {
	return o >= DescNullsFirst
}

func (o Order) tags() (present, null byte) {
	if o == AscNullsFirst || o == DescNullsFirst {
		return 1, 0
	}
	return 0, 1
}
