package deploy

// BulkResult summarizes a batch of operations.
type BulkResult struct {
	Results   []Result `json:"results"`
	Succeeded int      `json:"succeeded"`
	Failed    int      `json:"failed"`
}

// HasFailures returns true if any operation failed.
func (br *BulkResult) HasFailures() bool {
	return br.Failed > 0
}

// FailedResults returns only the failed results.
func (br *BulkResult) FailedResults() []Result {
	var failed []Result
	for _, r := range br.Results {
		if !r.Success {
			failed = append(failed, r)
		}
	}
	return failed
}
