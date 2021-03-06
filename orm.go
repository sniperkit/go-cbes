package cbes

import (
    "fmt"
    "time"
    "reflect"
    "encoding/json"
    "strconv"
)

type Orm struct {
    db *db
    tmpModel      interface{}
    tmpQuery      map[string]interface{}
    queryTemplate map[string]interface{}
}

// Execute builded query. If Aggregate() method is used this method will
// return the aggregations result from ElasticSearch
func (o *Orm) Do() []interface{} {
    jsonQuery, err := json.Marshal(o.tmpQuery)
    if err != nil {
        panic(err)
    }

    res := searchEs(string(jsonQuery))
    data := []interface{}{}

    if (res.TotalHits() == 0) {
        return data
    }

    if res.Aggregations != nil {
        aggJson, err := json.Marshal(res.Aggregations)
        if err != nil {
            panic(err)
        }

        items := make(map[string]interface{})
        err = json.Unmarshal(aggJson, &items)
        if err != nil {
            panic(err)
        }

        data = append(data, items)
        return data
    }

    for _, hit := range res.Hits.Hits {
        item := make(map[string]interface{})
        err := json.Unmarshal(*hit.Source, &item)
        if err != nil {
            panic(err)
        }

        data = append(data, setModel(o.tmpModel, item))
    }

    // reset query
    o.tmpQuery = make(map[string]interface{})

    return data
}

// Execute builded query and return elastic search count
func (o *Orm) Count() int64 {
    o.tmpQuery["size"] = 1

    jsonQuery, err := json.Marshal(o.tmpQuery)
    if err != nil {
        panic(err)
    }

    res := searchEs(string(jsonQuery))
    return res.TotalHits()
}

// Set the model witch you want to find
// If you don't set Limit Find ElasticSearch will use the default Limit (10)
func (o *Orm) Find(model interface{}) *Orm {
    // build a new orm object instance
    newOrm   := NewOrm()
    typeName := getModelName(model)
    _model, ok := modelsCache.get(typeName); if ok {
        newOrm.tmpModel = _model
    }

    // reset query
    newOrm.tmpQuery = make(map[string]interface{})

    // clone queryTemplate into tmpQuery
    _copy, err := json.Marshal(newOrm.queryTemplate)
    if err != nil {
        panic(err)
    }

    err = json.Unmarshal(_copy, &newOrm.tmpQuery)
    if err != nil {
        panic(err)
    }

    // set tmpQuery model type
    typeVal := newOrm.tmpQuery["query"].(map[string]interface{})["filtered"]
    typeVal = typeVal.(map[string]interface{})["query"]
    typeVal = typeVal.(map[string]interface{})["bool"]
    typeVal = typeVal.(map[string]interface{})["must"].([]interface{})[0]
    typeVal = typeVal.(map[string]interface{})["term"]
    typeVal = typeVal.(map[string]interface{})["_type"]
    typeVal.(map[string]interface{})["value"] = typeName

    return newOrm
}

// Set query for ElasticSearch
func (o *Orm) Where(query string) *Orm {
    if len(o.tmpQuery) == 0 {
        panic("You must declare Find() first!")
    }

    var q map[string]interface{}

    err := json.Unmarshal([]byte(query), &q)
    if err != nil {
        return o
    }

    o.tmpQuery["query"].(map[string]interface{})["filtered"].(map[string]interface{})["filter"] = q
    return o
}

// Pagination of results.  The from parameter defines the offset from the first result you want to fetch
// The Limit() function allows you to configure the maximum amount of hits to be returned. From() default is 0, and
// Limit() default is 10
func (o *Orm) From(from int) *Orm {
    if len(o.tmpQuery) == 0 {
        panic("You must declare Find() first!")
    }

    o.tmpQuery["from"] = from
    return o
}

// Set limit to Find() query in ElasticSearch. Max limit = 999999999
func (o *Orm) Limit(limit int) *Orm {
    if len(o.tmpQuery) == 0 {
        panic("You must declare Find() first!")
    }

    o.tmpQuery["size"] = limit
    return o
}

// Set ElasticSearch sort. direction: true = 'asc', false = 'desc'
func (o *Orm) Order(field string, direction bool) *Orm {
    if len(o.tmpQuery) == 0 {
        panic("You must declare Find() first!")
    }

    dir := "asc"
    if direction == false {
        dir = "desc"
    }

    sort := map[string]interface{}{
        field: map[string]interface{}{
            "order": dir,
        },
    }

    typeVal := o.tmpQuery["sort"].([]interface{})
    typeVal = append(typeVal, sort)

    o.tmpQuery["sort"] = typeVal
    return o
}

// Aggregate data using ElasticSearch
func (o *Orm) Aggregate(query string) *Orm {
    if len(o.tmpQuery) == 0 {
        panic("You must declare Find() first!")
    }

    var q map[string]interface{}

    err := json.Unmarshal([]byte(query), &q)
    if err != nil {
        return o
    }

    o.tmpQuery["aggs"] = q

    return o
}

// Create new document in CouchBase and Elasticsearch and return the created document.
func (o *Orm) Create(m interface{}) (interface{}, error) {
    t             := time.Now()
    timeFormatted := t.Format(time.RFC3339)
    model         := setModelDefaults(m)

    reflect.ValueOf(model).Elem().FieldByName("CreatedAt").SetString(timeFormatted)
    reflect.ValueOf(model).Elem().FieldByName("UpdatedAt").SetString(timeFormatted)
    reflect.ValueOf(model).Elem().FieldByName("TYPE").SetString(getModelName(model))

    id, err := createCB(model)
    if err != nil {
        return nil, fmt.Errorf("cbes.Create() CouchBase %s", err.Error())
    }

    err = createEs(id, model)
    if err != nil {
        return nil, fmt.Errorf("cbes.Create() ElasticSearch %s", err.Error())
    }

    reflect.ValueOf(model).Elem().FieldByName("ID").SetInt(id)

    return reflect.ValueOf(model).Elem().Interface(), nil
}

// Create a variadic of documents in CouchBase and ElasticSearch and return all successfully created documents
// even if an error occurs.
func (o *Orm) CreateEach(models ...interface{}) ([]interface{}, error) {
    data := make([]interface{}, 0)

    for _, model := range models {
        m, err := o.Create(model)
        if err != nil {
            return data, fmt.Errorf("cbes.CreateEach() CouchBase %s", err.Error())
        }

        data = append(data, m)
    }

    return data, nil
}

// Destroy a document in CouchBase and ElasticSearch.
// Returns all the deleted documents even if an error occurs.
func (o *Orm) Destroy(model interface{}, query string) ([]interface{}, error) {
    var err error
    res    := make([]interface{}, 0)

    models := o.Find(model).Where(query).Limit(999999999).Do()
    for _, m := range models {
        _m := reflect.ValueOf(m)
        id := _m.FieldByName("TYPE").String() + ":" + strconv.FormatInt(_m.FieldByName("ID").Int(), 10)

        err = destroyCB(id)
        if err != nil {
            return res, fmt.Errorf("cbes.Destroy() CouchBase %s", err.Error())
        }

        err = destroyES(id, model)
        if err != nil {
            return res, fmt.Errorf("cbes.Destroy() ElastiSearch %s", err.Error())
        }

        res = append(res, m)
    }

    return res, nil
}

// Update a document in CouchBase and ElasticSearch.
func (o *Orm) Update(model interface{}) error {
    var err error

    _model     := reflect.ValueOf(model).Elem()
    modelID    := _model.FieldByName("ID").Int()
    modelType  := _model.FieldByName("TYPE").String()
    id         := modelType + ":" + strconv.FormatInt(modelID, 10)
    updateTime := time.Now().Format(time.RFC3339)

    query := `{
        "query": {
            "bool": {
                "must": [
                    {
                        "term": {
                            "_id": "` + id + `"
                        }
                    }
                ]
            }
        }
    }`

    models := o.Find(model).Where(query).Do()
    if len(models) > 1 {
        return fmt.Errorf("cbes.Update() CouchBase - found to many models with ID %s", id)
    }

    if len(models) == 0 {
        return fmt.Errorf("Can't find model with ID %s", id)
    }

    _model.FieldByName("UpdatedAt").SetString(updateTime)

    setModel := _model.Interface()
    err = updateCB(id, setModel)
    if err != nil {
        return fmt.Errorf("cbes.Update() CouchBase %s", err.Error())
    }

    err = updateES(id, setModel)
    if err != nil {
        return fmt.Errorf("cbes.Update() ElasticSearch %s", err.Error())
    }

    return nil
}

// Get collection for the specified model
func (o *Orm) GetCollection(model interface{}) ([]interface{}, error) {
    res := []interface{}{}
    data, err := getByView(model)
    if err != nil {
        return res, err
    }

    for _, val := range data {
        doc := val.(map[string]interface{})["value"].(map[string]interface{})["doc"]
        res = append(res, setModel(model, doc))
    }

    return res, nil
}

// Get raw CouchBase collection view for the specified model
func (o *Orm) GetRawCollection(model interface{}) ([]interface{}, error) {
    data, err := getByView(model)
    if err != nil {
        return []interface{}{}, err
    }

    return data, nil
}

// Delete all documents and mapping from ElasticSearch for the given model and import them back
// from CouchBase. In the case of changing a mapping you should use this method so the new mapping
// will be applied
func (o *Orm) Reindex(model interface{}) error {
    var err error

    err = deleteMapping(model)
    if err != nil {
        return fmt.Errorf("cbes.Reindex() delete mapping %s", err.Error())
    }

    modelMapping := buildModelMapping(model)
    err = addMapping(modelMapping, getModelName(model))
    if err != nil {
        return fmt.Errorf("cbes.Reindex() put mapping %s", err.Error())
    }

    collection, err := o.GetCollection(model)
    if err != nil {
        return fmt.Errorf("cbes.Reindex() GetCollection %s", err.Error())
    }

    err = importModelsToEs(collection)
    if err != nil {
        return fmt.Errorf("cbes.Reindex() importModelsToEs %s", err.Error())
    }

    return nil
}

// Create a new ORM object with
func NewOrm() *Orm {
    return &Orm{
        tmpQuery: make(map[string]interface{}),
        queryTemplate: map[string]interface{}{
            "query": map[string]interface{}{
                "filtered": map[string]interface{}{
                    "query": map[string]interface{}{
                        "bool": map[string]interface{}{
                            "must": []interface{}{
                                map[string]interface{}{
                                    "term": map[string]interface{}{
                                        "_type": map[string]string{
                                            "value": "",
                                        },
                                    },
                                },
                            },
                        },
                    },
                    "filter": make(map[string]interface{}),
                },
            },
            "sort": []interface{}{},
        },
    }
}