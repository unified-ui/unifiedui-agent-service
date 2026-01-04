// Package docdb provides the document database type constants.
package docdb

// Type represents the type of document database.
type Type string

const (
	// TypeMongoDB represents a MongoDB database.
	TypeMongoDB Type = "mongodb"
	// TypeCosmosDB represents an Azure Cosmos DB database.
	TypeCosmosDB Type = "cosmosdb"
)
