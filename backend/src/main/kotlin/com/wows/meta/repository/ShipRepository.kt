package com.wows.meta.repository

import com.wows.meta.model.ShipEntity
import org.springframework.data.jpa.repository.JpaRepository
import org.springframework.stereotype.Repository

@Repository
interface ShipRepository : JpaRepository<ShipEntity, Long>
