package dynamo

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/hallgren/eventsourcing/core"
)

type DynamoEventStore struct {
	db        *dynamodb.DynamoDB
	tableName string
}

func New(sess *session.Session, tableName string) *DynamoEventStore {
	return &DynamoEventStore{
		db:        dynamodb.New(sess),
		tableName: tableName,
	}
}

func (store *DynamoEventStore) Save(events []core.Event) error {
	// Comprobar si hay eventos
	if len(events) == 0 {
		return nil
	}

	lastVersion, err := store.getLastVersion(events[0].AggregateID)
	if err != nil {
		// Manejar error al obtener la última versión
		return err
	}

	// Comprobar si la versión del evento es la siguiente versión esperada
	if core.Version(lastVersion)+1 != events[0].Version {
		return core.ErrConcurrency
	}

	for i := range events {
		// Obtener y actualizar el contador global de versión para cada evento
		globalVersion, err := store.getAndUpdateGlobalVersion()
		if err != nil {
			return err
		}

		// Establecer la versión global del evento
		events[i].GlobalVersion = globalVersion

		av, err := dynamodbattribute.MarshalMap(events[i])
		if err != nil {
			return err
		}

		// Preparar la expresión de condición para control de concurrencia
		condition := "attribute_not_exists(AggregateID) OR Version = :version"

		av["Version"] = &dynamodb.AttributeValue{
			N: aws.String(fmt.Sprintf("%d", events[i].Version)),
		}
		av["GlobalVersion"] = &dynamodb.AttributeValue{
			N: aws.String(fmt.Sprintf("%d", events[i].GlobalVersion)),
		}

		input := &dynamodb.PutItemInput{
			TableName: aws.String(store.tableName),
			Item:      av,
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":version": {N: aws.String(fmt.Sprintf("%d", events[i].Version-1))},
			},
			ConditionExpression: aws.String(condition),
		}

		_, err = store.db.PutItem(input)
		if err != nil {
			switch err.(type) {
			case *dynamodb.ConditionalCheckFailedException:
				return core.ErrConcurrency
			}

			return err
		}

	}
	return nil
}

func (store *DynamoEventStore) Get(
	ctx context.Context,
	id string,
	aggregateType string,
	afterVersion core.Version,
) (core.Iterator, error) {
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(store.tableName),
		KeyConditionExpression: aws.String("AggregateID = :v_id AND Version > :v_version"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v_id":      {S: aws.String(id)},
			":v_version": {N: aws.String(fmt.Sprintf("%d", afterVersion))},
		},
	}

	result, err := store.db.QueryWithContext(ctx, queryInput)
	if err != nil {
		return nil, err
	}

	return &iterator{
		items:   result.Items,
		current: -1,
	}, nil
}

func (store *DynamoEventStore) DB() *dynamodb.DynamoDB {
	return store.db
}

func (store *DynamoEventStore) getAndUpdateGlobalVersion() (core.Version, error) {
	// Define la clave primaria única para el ítem del contador global
	key := map[string]*dynamodb.AttributeValue{
		"AggregateID": {S: aws.String("GlobalVersionCounter")},
		"Version": {
			N: aws.String("0"),
		}, // Valor constante para cumplir con la estructura de clave primaria
	}

	// Expresión de actualización para incrementar GlobalVersion
	update := "SET GlobalVersion = if_not_exists(GlobalVersion, :start) + :inc"
	exprAttrValues := map[string]*dynamodb.AttributeValue{
		":inc":   {N: aws.String("1")}, // Incrementar en 1
		":start": {N: aws.String("0")}, // Valor inicial si no existe
	}

	input := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(store.tableName),
		Key:                       key,
		UpdateExpression:          aws.String(update),
		ExpressionAttributeValues: exprAttrValues,
		ReturnValues:              aws.String("UPDATED_NEW"), // Devuelve el valor actualizado
	}

	result, err := store.db.UpdateItem(input)
	if err != nil {
		return 0, err
	}

	// Obtener la versión global actualizada del resultado
	newVersion, err := strconv.ParseUint(*result.Attributes["GlobalVersion"].N, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Error al parsear GlobalVersion: %s", err)
	}

	return core.Version(newVersion), nil
}

func (store *DynamoEventStore) getLastVersion(aggregateID string) (core.Version, error) {
	// Definir el input para una consulta en DynamoDB
	input := &dynamodb.QueryInput{
		TableName:              aws.String(store.tableName),
		KeyConditionExpression: aws.String("AggregateID = :aggregateID"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":aggregateID": {
				S: aws.String(aggregateID),
			},
		},
		ScanIndexForward: aws.Bool(false), // Orden descendente
		Limit:            aws.Int64(1),    // Solo necesitamos el último evento
	}

	// Realizar la consulta
	result, err := store.db.Query(input)
	if err != nil {
		return 0, err
	}

	// Si no hay eventos, devuelve 0
	if len(result.Items) == 0 {
		return 0, nil
	}

	// Obtener el evento con la versión más alta
	var event core.Event
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &event)
	if err != nil {
		return 0, err
	}

	return event.Version, nil
}
