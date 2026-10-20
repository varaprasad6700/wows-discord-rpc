package com.wows.meta.model

import jakarta.persistence.Column
import jakarta.persistence.Entity
import jakarta.persistence.Id
import jakarta.persistence.Table
import java.time.Instant

@Entity
@Table(name = "ships")
class ShipEntity(
    @Id
    @Column(name = "ship_id", nullable = false)
    var shipId: Long = 0,

    @Column(name = "name", nullable = false)
    var name: String = "",

    @Column(name = "tier", nullable = false)
    var tier: Int = 1,

    @Column(name = "roman_tier", nullable = false, length = 10)
    var romanTier: String = "I",

    @Column(name = "ship_class", nullable = false, length = 50)
    var shipClass: String = "",

    @Column(name = "nation", nullable = false, length = 50)
    var nation: String = "",

    @Column(name = "icon_url", length = 1024)
    var iconUrl: String? = null,

    @Column(name = "last_updated")
    var lastUpdated: Instant = Instant.now()
)
