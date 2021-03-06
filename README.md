#go-cbes - Golang ORM for Couchbase & ElasticSearch
<a href="https://www.kreditech.com/" target="_blank" title="Kreditech"><img src="https://www.kreditech.com/wp-content/themes/kreditech/img/logo.svg" width="340" height="50" alt="Kreditech"/></a>

&copy; 2015 [Kreditech](http://www.kreditech.com/) / [Luigi Ilie Aron](https://github.com/aronluigi), [Laura Hreniucu](https://github.com/laura-h), [Robert Savu](https://github.com/r-savu), [Tiago Amorim](https://github.com/tiagoamorim85) & contributors

[![Doc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/Kreditech/go-cbes) [![Status](https://img.shields.io/badge/status-beta-orange.svg)]() [![Build Status](https://travis-ci.org/Kreditech/go-cbes.svg?branch=master)](https://travis-ci.org/Kreditech/go-cbes) [![Coverage Status](https://coveralls.io/repos/Kreditech/go-cbes/badge.svg?branch=master&service=github)](https://coveralls.io/github/Kreditech/go-cbes?branch=master) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/jmoiron/sqlx/master/LICENSE)

[more documentation on godoc.org](https://godoc.org/github.com/Kreditech/go-cbes)

go-cbes is a very fast ORM library for Golang that is using CouchBase and ElasticSearch as database. It uses idiomatic Go to operate on databases, implementing struct to database mapping and acts as a lightweight Go ORM framework. This library was designed to be supported by [Beego](http://beego.me/) or used as standalone library as well to find a good balance between functionality and performance.

Inspired from [sails-cbes](https://www.npmjs.com/package/sails-cbes), [Beego](http://beego.me/)


----------

##Requirements
 - ElasticSearch
 - Couchbase

Before using go-cbes make sure that you have installed and configure CouchBase and ElasticSearch. For CouchBase you need to create your bucket manually, go-cbes will create automatically the ElasticSearch Index.

----------

##Install
    go get github.com/Kreditech/go-cbes
    
----------


##Model
**All the numeric types must be 64 bit (int64, float64)!**
To define the mapping for ElasticSearch and the model default values, use the struct tags.
**At the moment default value works only for simple types not of arrays or objects!**

TTL - [expiration time](https://www.elastic.co/guide/en/elasticsearch/reference/1.4/mapping-ttl-field.html).

```
type TestModel struct {
    cbes.Model
    Name        string          `default:"TestName" type:"string" analyzer:"keyword" index:"analyzed"`
    LastName    string          `default:"Test Last Name" type:"string" analyzer:"standard" index:"analyzed"`
    Age         int64           `default:"25" type:"integer" analyzer:"standard" index:"not_analyzed"`
    Active      bool            `default:"true" type:"boolean"`
    Float       float64         `default:"12345.00" type:"float"`
    Int         int64           `default:"321" type:"long"`
    StringArray []string        `type:"string" analyzer:"keyword" index:"analyzed"`
    IntArray    []int64         `type:"integer" analyzer:"keyword" index:"analyzed"`
    FloatArray  []float64       `type:"float" analyzer:"keyword" index:"analyzed"`
    Interface   map[string]interface{} `type:"object" properties:"{'name':{'type':'object','enabled':false},'sid':{'type':'string','index':'not_analyzed'}}"`
    Nested      []interface{}   `type:"nested" properties:"{'first': {'type': 'string'}, 'last':{'type': 'string'}}"`
}

type TestModelTTL struct {
    cbes.Model
    Name        string          `default:"TestName" type:"string" analyzer:"keyword" index:"analyzed"`
    LastName    string          `default:"Test Last Name" type:"string" analyzer:"standard" index:"analyzed"`
    Age         int64           `default:"25" type:"integer" analyzer:"standard" index:"not_analyzed"`
    Active      bool            `default:"true" type:"boolean"`
    Float       float64         `default:"12345.00" type:"float"`
    Int         int64           `default:"321" type:"long"`
    StringArray []string        `type:"string" analyzer:"keyword" index:"analyzed"`
    IntArray    []int64         `type:"integer" analyzer:"keyword" index:"analyzed"`
    FloatArray  []float64       `type:"float" analyzer:"keyword" index:"analyzed"`
    Interface   map[string]interface{} `type:"object" properties:"{'name':{'type':'object','enabled':false},'sid':{'type':'string','index':'not_analyzed'}}"`
    Nested      []interface{}   `type:"nested" properties:"{'first': {'type': 'string'}, 'last':{'type': 'string'}}"`
    ttl         int64           `ttl:"25"` //ttl in seconds
}
```
##RegisterModel
Before register the db we need to register our models.
```
err := cbes.RegisterModel(new(TestModel), new(TestModelTTL))

if err != nil {
    t.Fatal(err)
}
```
##RegisterDataBase
After we register our models we can register our db
```
settings := new(cbes.Settings)
settings.ElasticSearch.Urls = []string{"http://192.168.33.10:9200"}
settings.ElasticSearch.Index = "testindex"
settings.ElasticSearch.NumberOfShards = 5
settings.ElasticSearch.NumberOfReplicas = 1
settings.ElasticSearch.RefreshInterval = "10ms"
settings.ElasticSearch.CheckOnStartup = true

settings.CouchBase.Host = "192.168.33.10:8091"
settings.CouchBase.UserName = "root"
settings.CouchBase.Pass = "root123"

bucket := new(cbes.Bucket)
bucket.Name = "test"
bucket.Pass = ""
bucket.OperationTimeout = 5 // seconds

settings.CouchBase.Bucket = bucket

viewsOptions := new(cbes.ViewsOptions)
viewsOptions.UpdateInterval = 5000
viewsOptions.UpdateMinChanges = 5
viewsOptions.ReplicaUpdateMinChanges = 5

settings.CouchBase.ViewsOptions = viewsOptions

err := cbes.RegisterDataBase(settings)
if err != nil {
    t.Fatal(err)
}
```
##Build connection
```
    o := cbes.NewOrm()
```
##Create()
Returns the created document.
```
var err error
o := cbes.NewOrm()

resModel, err := o.Create(&testModel)
if err != nil {
    t.Fatal(err)
}

_ = resModel.(TestModel)

resModel, err = o.Create(&testModelTtl)
if err != nil {
    t.Fatal(err)
}

_ = resModel.(TestModelTTL)
```
##CreateEach()
Returns all successfully created documents even if an error occurs.
```
var err error
var models []interface{}
var modelsTtl []interface{}
o := cbes.NewOrm()

for i := 0; i < 10; i++ {
    models = append(models, &testModel)
    modelsTtl = append(modelsTtl, &testModelTtl)
}

createdModels, err := o.CreateEach(models...)
if err != nil {
    t.Fatal(err)
}

for _, m := range createdModels {
    _ = m.(TestModel)
}

createdModels, err = o.CreateEach(modelsTtl...)
if err != nil {
    t.Fatal(err)
}

for _, m := range createdModels {
    _ = m.(TestModelTTL)
}
```
##Count()
```
o := cbes.NewOrm()
q := `{
    "query": {
        "bool": {
            "must": [
                {
                    "term": {
                        "Name": "` + testModel.Name + `"
                    }
                }
            ]
        }
    }
}`

count := o.Find(&testModel).Where(q).Count()
if count != 11 {
    t.Fatalf("Wrong Count")
}
```
##Update()
```
o := cbes.NewOrm()
q := `{
    "query": {
        "bool": {
            "must": [
                {
                    "term": {
                        "Name": "` + testModel.Name + `"
                    }
                }
            ]
        }
    }
}`

res := o.Find(&testModel).Where(q).Do()
if len(res) < 1 {
    t.Fatalf("No results found")
}

for i := 0; i < len(res); i++ {
    m := res[i].(TestModel)

    newAge := i * 100
    m.Age = int64(newAge)
    m.StringArray = []string{}

    err := o.Update(&m)
    if err != nil {
         t.Fatal(err)
    }
}
```
##Destroy()
Delete by query.
Returns all successfully deleted documents even if an error occurs.
```
o := cbes.NewOrm()
q := `{
    "query": {
        "bool": {
            "must": [
                {
                    "term": {
                        "Age": 300
                    }
                }
            ]
        }
    }
}`

affected, err := o.Destroy(&testModel, q)
if err != nil {
    t.Fatal(err)
}

if len(affected) == 0 {
    t.Fatalf("Objects not destroyed")
}

for _, deletedModel := range affected {
    _ = deletedModel.(TestModel)
}

affected, err = o.Destroy(&testModel, "")
if err != nil {
    t.Fatal(err)
}

if len(affected) == 0 {
    t.Fatalf("Objects not destroyed")
}

for _, deletedModel := range affected {
    _ = deletedModel.(TestModel)
}
```
##Find(), Where(), Do() 
```
o := cbes.NewOrm()
q := `{
    "query": {
        "bool": {
            "must": [
                {
                    "term": {
                        "Name": "` + testModel.Name + `"
                    }
                }
            ]
        }
    }
}`

res := o.Find(&testModel).Where(q).Do()
m := res[0].(TestModel)
```
##Order()
```
res := o.Find(&testModel).Where(q).Order("ID", true).Do()
```
##Limit()
```
res := o.Find(&testModel).Where(q).Limit(1).Do()
```
##From()
```
res := o.Find(&testModel).Where(q).Limit(2).From(3).Do()
```
##Aggregate()
```
o := cbes.NewOrm()
q := `{
    "query": {
        "bool": {
            "must": [
                {
                    "term": {
                        "Name": "` + testModel.Name + `"
                    }
                }
            ]
        }
    }
}`

aggQuery := `{
      "test_agg": {
          "terms": {
              "field": "Age"
          },
          "aggs":{
              "IDS_max":{
                  "max": {
                      "field": "Age"
                  }
              }
          }
      }
  }
`

res := o.Find(&testModel).Where(q).Aggregate(aggQuery).Do()
```
##GetCollection()
We recommend you to use this method every time you want to get all models of one type. This method is using CouchBase views in order to return huge amounts of data very fast.
```
o := cbes.NewOrm()
collection, err := o.GetCollection(&testModel)
if err != nil {
    t.Fatal(err)
}
m := collection[0].(TestModel)
```
##GetRawCollection()
```
o := cbes.NewOrm()
collection, err := o.GetRawCollection(&testModel)
if err != nil {
    t.Fatal(err)
}
```
##Reindex()
Every time you edit/add/remove the model mapping you need to use the reindex() method. This method will remove the mapping for the specified model from ElasticSearch and generate a new one + it will reimport all entities for the specified model from CouchBase to ElasticSearch. 
```
o := cbes.NewOrm()

err := o.Reindex(&testModel)
if err != nil {
    t.Fatal(err)
}

time.Sleep(30 * time.Millisecond)

q := `{
    "query": {
        "bool": {
            "must": [
                {
                    "term": {
                        "Name": "` + testModel.Name + `"
                    }
                }
            ]
        }
    }
}`

res := o.Find(&testModel).Where(q).Do()
m := res[0].(TestModel)
```
