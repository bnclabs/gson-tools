package main

import "bytes"

import "github.com/prataprc/gson"

type sgmts []string

type jptrs []*gson.Jsonpointer

func (ptrs jptrs) Len() int {
	return len(ptrs)
}

func (ptrs jptrs) Less(i, j int) bool {
	return len(ptrs[i].Segments()) < len(ptrs[j].Segments())
}

func (ptrs jptrs) Swap(i, j int) {
	ptrs[i], ptrs[j] = ptrs[j], ptrs[i]
}

func (ptrs jptrs) filterAppend() jptrs {
	nptrs := make(jptrs, 0, len(ptrs))
	for _, ptr := range ptrs {
		segments := ptr.Segments()
		ln := len(segments)
		if ln == 0 || bytes.Compare(segments[ln-1], []byte("-")) == 0 {
			nptrs = append(nptrs, ptr)
		}
	}
	return nptrs
}

//func setcontainer(s sgmts, doc, replica interface{}) {
//    switch get(s, doc).(type) {
//    case []interface{}:
//        set(s, replica, []interface{}{})
//    case map[string]interface{}:
//        set(s, replica, map[string]interface{}{})
//    default:
//        // does not point to a container
//    }
//}
