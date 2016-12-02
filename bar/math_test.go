package bar

import "testing"

func TestMin(t *testing.T) {
	if Min(1,2)!=1{
		t.Error("error")
	}
	if Min(0,-1)!=0{
		t.Error("error")
	}
}