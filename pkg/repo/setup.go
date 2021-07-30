package repo

func (c Client) Setup(rootVersion int64, rootSha512 string) error {
	return c.tufClient.Setup(rootVersion, rootSha512)
}
