package persister

func (sp *Snapshot) IsEmptySnap() bool {
	return sp.LastIndex == 0
}
