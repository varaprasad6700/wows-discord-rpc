package com.wows.meta.dto

import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty

@JsonIgnoreProperties(ignoreUnknown = true)
data class WargamingApiResponse(
    val status: String = "",
    val meta: WargamingMeta = WargamingMeta(),
    val data: Map<String, WargamingShipData>? = null
)

@JsonIgnoreProperties(ignoreUnknown = true)
data class WargamingMeta(
    val count: Int = 0,
    @JsonProperty("page_total")
    val pageTotal: Int = 1,
    val page: Int = 1
)

@JsonIgnoreProperties(ignoreUnknown = true)
data class WargamingShipData(
    @JsonProperty("ship_id")
    val shipId: Long = 0,
    val name: String = "",
    val tier: Int = 1,
    val type: String = "",
    val nation: String = "",
    val images: WargamingShipImages? = null
)

@JsonIgnoreProperties(ignoreUnknown = true)
data class WargamingShipImages(
    val small: String? = null,
    val medium: String? = null,
    val icon: String? = null
)
