package com.wows.meta.dto

data class ShipResponseDto(
    val shipId: Long,
    val name: String,
    val tier: Int,
    val romanTier: String,
    val shipClass: String,
    val nation: String,
    val iconUrl: String?
)
