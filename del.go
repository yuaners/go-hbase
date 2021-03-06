package hbase

import (
	"github.com/c4pt0r/go-hbase/proto"
	pb "github.com/golang/protobuf/proto"

	"fmt"
	"math"
	"strings"
)

type Delete struct {
	Row         []byte
	Families    set
	FamilyQuals map[string]set
	Ts          map[string]uint64
}

func NewDelete(row []byte) *Delete {
	return &Delete{
		Row:         row,
		Families:    newSet(),
		FamilyQuals: make(map[string]set),
		Ts:          make(map[string]uint64),
	}
}

func (d *Delete) AddString(famqual string) error {
	parts := strings.Split(famqual, ":")

	if len(parts) > 2 {
		return fmt.Errorf("Too many colons were found in the family:qualifier string. '%s'", famqual)
	} else if len(parts) == 2 {
		d.AddStringColumn(parts[0], parts[1])
	} else {
		d.AddStringFamily(famqual)
	}

	return nil
}

func (d *Delete) GetRow() []byte {
	return d.Row
}

func (d *Delete) AddStringColumn(family, qual string) {
	d.AddColumn([]byte(family), []byte(qual))
}

func (d *Delete) AddStringFamily(family string) {
	d.AddFamily([]byte(family))
}

func (d *Delete) AddColumn(family, qual []byte) {
	d.AddFamily(family)
	d.FamilyQuals[string(family)].add(string(qual))
}

func (d *Delete) AddFamily(family []byte) {
	d.Families.add(string(family))
	if _, ok := d.FamilyQuals[string(family)]; !ok {
		d.FamilyQuals[string(family)] = newSet()
	}
}

func (d *Delete) AddColumnWithTimestamp(family, qual []byte, ts uint64) {
	d.AddColumn(family, qual)
	k := string(family) + ":" + string(qual)
	d.Ts[k] = ts
}

func (d *Delete) ToProto() pb.Message {
	del := &proto.MutationProto{
		Row:        d.Row,
		MutateType: proto.MutationProto_DELETE.Enum(),
	}

	for family, _ := range d.Families {
		cv := &proto.MutationProto_ColumnValue{
			Family:         []byte(family),
			QualifierValue: make([]*proto.MutationProto_ColumnValue_QualifierValue, 0),
		}

		if len(d.FamilyQuals[family]) == 0 {
			cv.QualifierValue = append(cv.QualifierValue, &proto.MutationProto_ColumnValue_QualifierValue{
				Qualifier:  nil,
				Timestamp:  pb.Uint64(uint64(math.MaxInt64)),
				DeleteType: proto.MutationProto_DELETE_FAMILY.Enum(),
			})
		}

		for qual, _ := range d.FamilyQuals[family] {
			v := &proto.MutationProto_ColumnValue_QualifierValue{
				Qualifier:  []byte(qual),
				Timestamp:  pb.Uint64(uint64(math.MaxInt64)),
				DeleteType: proto.MutationProto_DELETE_MULTIPLE_VERSIONS.Enum(),
			}
			tsKey := string(family) + ":" + string(qual)
			if ts, ok := d.Ts[tsKey]; ok {
				v.Timestamp = pb.Uint64(ts)
				v.DeleteType = proto.MutationProto_DELETE_ONE_VERSION.Enum()
			}
			cv.QualifierValue = append(cv.QualifierValue, v)
		}

		del.ColumnValue = append(del.ColumnValue, cv)
	}

	return del
}
