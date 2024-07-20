package persister

var EmptyState = &State{}

func IsEmptyState(a *State) bool {
	return isEmptyState(a, EmptyState)
}

func isEmptyState(a, b *State) bool {
	return a.VotedFor == b.VotedFor && a.CurrentTerm == b.CurrentTerm
}

func MustSync(st, prevst *State, entsnum int) bool {
	return entsnum != 0 || st.VotedFor != prevst.VotedFor || st.CurrentTerm != prevst.CurrentTerm
}
