SELECT pk.id
FROM public_key pk
    JOIN public_key_to_tag pkt_internal ON pkt_internal.public_key_id = pk.id
    JOIN tag t_internal ON t_internal.id = pkt_internal.tag_id
    AND t_internal.name = 'internal'
    LEFT JOIN public_key_to_tag pkt_partner ON pkt_partner.public_key_id = pk.id
    LEFT JOIN tag t_partner ON t_partner.id = pkt_partner.tag_id
    AND t_partner.name = 'partner'
    JOIN public_key_to_tag pkt_v2 ON pkt_v2.public_key_id = pk.id
    JOIN tag t_v2 ON t_v2.id = pkt_v2.tag_id
    AND t_v2.name LIKE 'v2-9._'
    LEFT JOIN public_key_to_tag pkt_legacy ON pkt_legacy.public_key_id = pk.id
    LEFT JOIN tag t_legacy ON t_legacy.id = pkt_legacy.tag_id
    AND t_legacy.name = 'legacy'
    LEFT JOIN public_key_to_tag pkt_deprecated ON pkt_deprecated.public_key_id = pk.id
    LEFT JOIN tag t_deprecated ON t_deprecated.id = pkt_deprecated.tag_id
    AND t_deprecated.name = 'deprecated'
    LEFT JOIN public_key_to_tag pkt_temp ON pkt_temp.public_key_id = pk.id
    LEFT JOIN tag t_temp ON t_temp.id = pkt_temp.tag_id
    AND t_temp.name LIKE '_temp_'
    LEFT JOIN public_key_to_tag pkt_tier1 ON pkt_tier1.public_key_id = pk.id
    LEFT JOIN tag t_tier1 ON t_tier1.id = pkt_tier1.tag_id
    AND t_tier1.name = 'tier-1'
    LEFT JOIN public_key_to_tag pkt_mission ON pkt_mission.public_key_id = pk.id
    LEFT JOIN tag t_mission ON t_mission.id = pkt_mission.tag_id
    AND t_mission.name = 'mission-critical'
    JOIN public_key_to_tag pkt_backend ON pkt_backend.public_key_id = pk.id
    JOIN tag t_backend ON t_backend.id = pkt_backend.tag_id
    AND t_backend.name = 'backend'
    LEFT JOIN public_key_to_tag pkt_python ON pkt_python.public_key_id = pk.id
    LEFT JOIN tag t_python ON t_python.id = pkt_python.tag_id
    AND t_python.name = 'python'
    LEFT JOIN public_key_to_tag pkt_go ON pkt_go.public_key_id = pk.id
    LEFT JOIN tag t_go ON t_go.id = pkt_go.tag_id
    AND t_go.name LIKE 'go_'
    LEFT JOIN public_key_to_tag pkt_aws ON pkt_aws.public_key_id = pk.id
    LEFT JOIN tag t_aws ON t_aws.id = pkt_aws.tag_id
    AND t_aws.name = 'aws'
    LEFT JOIN public_key_to_tag pkt_azure ON pkt_azure.public_key_id = pk.id
    LEFT JOIN tag t_azure ON t_azure.id = pkt_azure.tag_id
    AND t_azure.name = 'azure'
    LEFT JOIN public_key_to_tag pkt_gcp ON pkt_gcp.public_key_id = pk.id
    LEFT JOIN tag t_gcp ON t_gcp.id = pkt_gcp.tag_id
    AND t_gcp.name = 'gcp'
    LEFT JOIN public_key_to_tag pkt_us ON pkt_us.public_key_id = pk.id
    LEFT JOIN tag t_us ON t_us.id = pkt_us.tag_id
    AND t_us.name = 'us'
    LEFT JOIN public_key_to_tag pkt_eu ON pkt_eu.public_key_id = pk.id
    LEFT JOIN tag t_eu ON t_eu.id = pkt_eu.tag_id
    AND t_eu.name = 'eu'
WHERE (
        t_internal.id IS NOT NULL
        OR t_partner.id IS NOT NULL
    )
    AND t_v2.id IS NOT NULL
    AND NOT (
        (
            t_legacy.id IS NOT NULL
            OR t_deprecated.id IS NOT NULL
            OR t_temp.id IS NOT NULL
        )
        AND (
            t_tier1.id IS NULL
            AND t_mission.id IS NULL
        )
    )
    AND t_backend.id IS NOT NULL
    AND (
        t_python.id IS NOT NULL
        OR t_go.id IS NOT NULL
    )
    AND NOT (
        (
            t_aws.id IS NOT NULL
            OR t_azure.id IS NOT NULL
            OR t_gcp.id IS NOT NULL
        )
        AND (
            t_us.id IS NOT NULL
            OR t_eu.id IS NOT NULL
        )
    );

/* (internal | partner) & v2-9.* & !((legacy | deprecated | *temp*) & !(tier-1 | mission-critical)) & backend & (python | go*) & !((aws | azure | gcp) & (us | eu)) */
SELECT "id"
FROM "public_key"
WHERE (
        id IN (
            /* internal | partner */
            SELECT "id"
            FROM "public_key"
            WHERE (
                    id IN (
                        /* internal */
                        SELECT "public_key_id"
                        FROM "public_key_to_tag"
                            JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                        WHERE (tag.name = 'internal')
                    )
                )
                OR (
                    id IN (
                        /* partner */
                        SELECT "public_key_id"
                        FROM "public_key_to_tag"
                            JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                        WHERE (tag.name = 'partner')
                    )
                )
        )
    )
    AND (
        id IN (
            /* v2-9.* */
            SELECT "public_key_id"
            FROM "public_key_to_tag"
                JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
            WHERE (tag.name LIKE 'v2!-9._' ESCAPE '!')
        )
    )
    AND (
        id IN (
            /* !((legacy | deprecated | *temp*) & !(tier-1 | mission-critical)) */
            SELECT "id"
            FROM "public_key"
            WHERE (
                    id NOT IN (
                        /* (legacy | deprecated | *temp*) & !(tier-1 | mission-critical) */
                        SELECT "id"
                        FROM "public_key"
                        WHERE (
                                id IN (
                                    /* legacy | deprecated | *temp* */
                                    SELECT "id"
                                    FROM "public_key"
                                    WHERE (
                                            id IN (
                                                /* legacy */
                                                SELECT "public_key_id"
                                                FROM "public_key_to_tag"
                                                    JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                                                WHERE (tag.name = 'legacy')
                                            )
                                        )
                                        OR (
                                            id IN (
                                                /* deprecated */
                                                SELECT "public_key_id"
                                                FROM "public_key_to_tag"
                                                    JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                                                WHERE (tag.name = 'deprecated')
                                            )
                                        )
                                        OR (
                                            id IN (
                                                /* *temp* */
                                                SELECT "public_key_id"
                                                FROM "public_key_to_tag"
                                                    JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                                                WHERE (tag.name LIKE '_temp_' ESCAPE '!')
                                            )
                                        )
                                )
                            )
                            AND (
                                id IN (
                                    /* !(tier-1 | mission-critical) */
                                    SELECT "id"
                                    FROM "public_key"
                                    WHERE (
                                            id NOT IN (
                                                /* tier-1 | mission-critical */
                                                SELECT "id"
                                                FROM "public_key"
                                                WHERE (
                                                        id IN (
                                                            /* tier-1 */
                                                            SELECT "public_key_id"
                                                            FROM "public_key_to_tag"
                                                                JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                                                            WHERE (tag.name = 'tier!-1')
                                                        )
                                                    )
                                                    OR (
                                                        id IN (
                                                            /* mission-critical */
                                                            SELECT "public_key_id"
                                                            FROM "public_key_to_tag"
                                                                JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                                                            WHERE (tag.name = 'mission!-critical')
                                                        )
                                                    )
                                            )
                                        )
                                )
                            )
                    )
                )
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
    )
    AND (
        id IN (
            /* python | go* */
            SELECT "id"
            FROM "public_key"
            WHERE (
                    id IN (
                        /* python */
                        SELECT "public_key_id"
                        FROM "public_key_to_tag"
                            JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                        WHERE (tag.name = 'python')
                    )
                )
                OR (
                    id IN (
                        /* go* */
                        SELECT "public_key_id"
                        FROM "public_key_to_tag"
                            JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                        WHERE (tag.name LIKE 'go_' ESCAPE '!')
                    )
                )
        )
    )
    AND (
        id IN (
            /* !((aws | azure | gcp) & (us | eu)) */
            SELECT "id"
            FROM "public_key"
            WHERE (
                    id NOT IN (
                        /* (aws | azure | gcp) & (us | eu) */
                        SELECT "id"
                        FROM "public_key"
                        WHERE (
                                id IN (
                                    /* aws | azure | gcp */
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
                                                /* azure */
                                                SELECT "public_key_id"
                                                FROM "public_key_to_tag"
                                                    JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                                                WHERE (tag.name = 'azure')
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
                                    /* us | eu */
                                    SELECT "id"
                                    FROM "public_key"
                                    WHERE (
                                            id IN (
                                                /* us */
                                                SELECT "public_key_id"
                                                FROM "public_key_to_tag"
                                                    JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                                                WHERE (tag.name = 'us')
                                            )
                                        )
                                        OR (
                                            id IN (
                                                /* eu */
                                                SELECT "public_key_id"
                                                FROM "public_key_to_tag"
                                                    JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id)
                                                WHERE (tag.name = 'eu')
                                            )
                                        )
                                )
                            )
                    )
                )
        )
    )