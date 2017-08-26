package credential

type AzureStorage struct {
	generic
}

func (c AzureStorage) Account() string {
	return c.Data[AzureStorageAccount]
}

func (c AzureStorage) Key() string {
	return c.Data[AzureStorageKey]
}
