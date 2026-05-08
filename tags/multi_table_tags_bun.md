# Tag Matcher
"(user1 | user2) & !hagebau"

# Tabellen
public_keys:
- id
- key

tags_to_pk:
- pk_id
- tag_id

tags
- id
- name

# SQL Request
```sql
SELECT DISTINCT pk.*
FROM public_keys AS pk
JOIN tags_to_pk AS ttpk ON ttpk.pk_id = pk.id
JOIN tags AS t ON t.id = ttpk.tag_id
WHERE pk.id IN (
    SELECT ttpk1.pk_id
    FROM tags_to_pk AS ttpk1
    JOIN tags AS t1 ON t1.id = ttpk1.tag_id
    WHERE t1.name IN ('user1', 'user2')
)
AND pk.id NOT IN (
    SELECT ttpk2.pk_id
    FROM tags_to_pk AS ttpk2
    JOIN tags AS t2 ON t2.id = ttpk2.tag_id
    WHERE t2.name = 'hagebau'
);
```

# Bun implementation
```go
var publicKeys []PublicKey

subInclude := db.NewSelect().
	Model((*TagsToPk)(nil)).
	Column("pk_id").
	Join("JOIN tags AS t1 ON t1.id = tags_to_pk.tag_id").
	Where("t1.name IN (?)", bun.In([]string{"user1", "user2"}))

subExclude := db.NewSelect().
	Model((*TagsToPk)(nil)).
	Column("pk_id").
	Join("JOIN tags AS t2 ON t2.id = tags_to_pk.tag_id").
	Where("t2.name = ?", "hagebau")

err := db.NewSelect().
	Model(&publicKeys).
	Distinct().
	Join("JOIN tags_to_pk AS ttpk ON ttpk.pk_id = public_key.id").
	Join("JOIN tags AS t ON t.id = ttpk.tag_id").
	Where("public_key.id IN (?)", subInclude).
	Where("public_key.id NOT IN (?)", subExclude).
	Scan(ctx)
```
