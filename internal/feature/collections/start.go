package collection

// StartWith is what the program opens when no -workspace folder is given and
// no collection is remembered.
//
// Why an enum and not a nil Sample: nil already means GitHubSample, and a
// flag beside Sample would be two fields for one decision (rejected: a
// NoSample bool). Revisit when a third start is wanted. The build decides it;
// it is never saved.
type StartWith int

const (
	// StartWithSample opens the start collection: the client config's tree,
	// else Sample, unsaved and in memory. It is the zero value, so a caller
	// that says nothing keeps the default build's behavior.
	StartWithSample StartWith = iota
	// StartWithNothing opens no collection: the tree shows "No collection is
	// open" and one blank tab belongs to none. Sample and Clients are not
	// read.
	StartWithNothing
)
