-- /* prod */
SELECT "public_key_id"
FROM "public_key_to_tag"
    JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
WHERE (tag.name = 'prod');

-- /* v* */
SELECT "public_key_id"
FROM "public_key_to_tag"
    JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
WHERE (tag.name LIKE 'v_' ESCAPE '!');

-- /* api-** */
SELECT "public_key_id"
FROM "public_key_to_tag"
    JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
WHERE (tag.name LIKE 'api!-%' ESCAPE '!');

-- /* !deprecated */
SELECT "id"
FROM "public_key"
WHERE (
        id NOT IN (
            /* deprecated */
            SELECT "public_key_id"
            FROM "public_key_to_tag"
                JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
            WHERE (tag.name = 'deprecated')
        )
    );
    
-- /* golang & backend */
SELECT "id"
FROM "public_key"
WHERE (
        id IN (
            /* golang */
            SELECT "public_key_id"
            FROM "public_key_to_tag"
                JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
            WHERE (tag.name = 'golang')
        )
    )
    AND (
        id IN (
            /* backend */
            SELECT "public_key_id"
            FROM "public_key_to_tag"
                JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
            WHERE (tag.name = 'backend')
        )
    );
    
-- /* ios | android */
SELECT "id"
FROM "public_key"
WHERE (
        id IN (
            /* ios */
            SELECT "public_key_id"
            FROM "public_key_to_tag"
                JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
            WHERE (tag.name = 'ios')
        )
    )
    OR (
        id IN (
            /* android */
            SELECT "public_key_id"
            FROM "public_key_to_tag"
                JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
            WHERE (tag.name = 'android')
        )
    );
    
-- /* (aws | gcp) & !legacy */
SELECT "id"
FROM "public_key"
WHERE (
        id IN (
            /* aws | gcp */
            SELECT "id"
            FROM "public_key"
            WHERE (
                    id IN (
                        /* aws */
                        SELECT "public_key_id"
                        FROM "public_key_to_tag"
                            JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                        WHERE (tag.name = 'aws')
                    )
                )
                OR (
                    id IN (
                        /* gcp */
                        SELECT "public_key_id"
                        FROM "public_key_to_tag"
                            JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                        WHERE (tag.name = 'gcp')
                    )
                )
        )
    )
    AND (
        id IN (
            /* !legacy */
            SELECT "id"
            FROM "public_key"
            WHERE (
                    id NOT IN (
                        /* legacy */
                        SELECT "public_key_id"
                        FROM "public_key_to_tag"
                            JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                        WHERE (tag.name = 'legacy')
                    )
                )
        )
    );
    
-- /* auth & (**-admin | super-**) */
SELECT "id"
FROM "public_key"
WHERE (
        id IN (
            /* auth */
            SELECT "public_key_id"
            FROM "public_key_to_tag"
                JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
            WHERE (tag.name = 'auth')
        )
    )
    AND (
        id IN (
            /* **-admin | super-** */
            SELECT "id"
            FROM "public_key"
            WHERE (
                    id IN (
                        /* **-admin */
                        SELECT "public_key_id"
                        FROM "public_key_to_tag"
                            JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                        WHERE (tag.name LIKE '%!-admin' ESCAPE '!')
                    )
                )
                OR (
                    id IN (
                        /* super-** */
                        SELECT "public_key_id"
                        FROM "public_key_to_tag"
                            JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                        WHERE (tag.name LIKE 'super!-%' ESCAPE '!')
                    )
                )
        )
    );
    
-- /* !(test | stage) & prod */
SELECT "id"
FROM "public_key"
WHERE (
        id IN (
            /* !(test | stage) */
            SELECT "id"
            FROM "public_key"
            WHERE (
                    id NOT IN (
                        /* test | stage */
                        SELECT "id"
                        FROM "public_key"
                        WHERE (
                                id IN (
                                    /* test */
                                    SELECT "public_key_id"
                                    FROM "public_key_to_tag"
                                        JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                                    WHERE (tag.name = 'test')
                                )
                            )
                            OR (
                                id IN (
                                    /* stage */
                                    SELECT "public_key_id"
                                    FROM "public_key_to_tag"
                                        JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                                    WHERE (tag.name = 'stage')
                                )
                            )
                    )
                )
        )
    )
    AND (
        id IN (
            /* prod */
            SELECT "public_key_id"
            FROM "public_key_to_tag"
                JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
            WHERE (tag.name = 'prod')
        )
    );
    
-- /* ** */
SELECT "public_key_id"
FROM "public_key_to_tag"
    JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
WHERE (tag.name LIKE '%' ESCAPE '!');

-- /* * */
SELECT "public_key_id"
FROM "public_key_to_tag"
    JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
WHERE (tag.name LIKE '_' ESCAPE '!');
