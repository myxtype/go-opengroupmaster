package handler

func parseWordCloudBlacklistBatch(raw string) []string {
	return parseBannedWordsBatch(raw)
}
