package com.sims.dormitory.service.impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.common.response.ErrorCode;
import com.sims.dormitory.common.exception.BusinessException;
import com.sims.dormitory.model.dto.AlertDTO;
import com.sims.dormitory.model.entity.DormAlert;
import com.sims.dormitory.repository.DormAlertMapper;
import com.sims.dormitory.service.DormitoryAlertService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.List;
import java.util.stream.Collectors;

@Service
public class DormitoryAlertServiceImpl implements DormitoryAlertService {

    private static final Logger log = LoggerFactory.getLogger(DormitoryAlertServiceImpl.class);

    private final DormAlertMapper alertMapper;

    public DormitoryAlertServiceImpl(DormAlertMapper alertMapper) {
        this.alertMapper = alertMapper;
    }

    @Override
    @Transactional
    public DormAlert createAlert(DormAlert alert) {
        // TODO: implement alert creation with deduplication
        // 1. Check alert cooldown
        // 2. Insert alert record
        // 3. Send Kafka notification
        log.info("Creating alert: type={}, buildingId={}", alert.getAlertType(), alert.getBuildingId());
        alert.setAcknowledged(false);
        alert.setCreatedAt(LocalDateTime.now());
        alertMapper.insert(alert);
        return alert;
    }

    @Override
    @Transactional
    public void acknowledgeAlert(Long id, String acknowledgedBy) {
        // TODO: implement alert acknowledgement
        DormAlert alert = alertMapper.selectById(id);
        if (alert == null) {
            throw new BusinessException(ErrorCode.NOT_FOUND);
        }
        alert.setAcknowledged(true);
        alert.setAcknowledgedBy(acknowledgedBy);
        alert.setAcknowledgedAt(LocalDateTime.now());
        alertMapper.updateById(alert);
        log.info("Alert acknowledged: id={}, by={}", id, acknowledgedBy);
    }

    @Override
    public Page<AlertDTO> getAlerts(Long buildingId, String alertType, Boolean acknowledged,
                                    int page, int size) {
        // TODO: implement paginated alert query with filters
        Page<DormAlert> alertPage = new Page<>(page, size);
        LambdaQueryWrapper<DormAlert> wrapper = new LambdaQueryWrapper<>();
        if (buildingId != null) {
            wrapper.eq(DormAlert::getBuildingId, buildingId);
        }
        if (alertType != null) {
            wrapper.eq(DormAlert::getAlertType, alertType);
        }
        if (acknowledged != null) {
            wrapper.eq(DormAlert::getAcknowledged, acknowledged);
        }
        wrapper.orderByDesc(DormAlert::getCreatedAt);

        Page<DormAlert> result = alertMapper.selectPage(alertPage, wrapper);

        List<AlertDTO> dtoList = result.getRecords().stream()
                .map(this::toDTO)
                .collect(Collectors.toList());

        Page<AlertDTO> dtoPage = new Page<>(result.getCurrent(), result.getSize(), result.getTotal());
        dtoPage.setRecords(dtoList);
        return dtoPage;
    }

    @Override
    public long getAlertCount(Long buildingId, Boolean acknowledged) {
        LambdaQueryWrapper<DormAlert> wrapper = new LambdaQueryWrapper<>();
        if (buildingId != null) {
            wrapper.eq(DormAlert::getBuildingId, buildingId);
        }
        if (acknowledged != null) {
            wrapper.eq(DormAlert::getAcknowledged, acknowledged);
        }
        return alertMapper.selectCount(wrapper);
    }

    private AlertDTO toDTO(DormAlert alert) {
        return new AlertDTO(
                alert.getId(),
                alert.getBuildingId(),
                alert.getAlertType(),
                alert.getMessage(),
                alert.getDetails(),
                alert.getAcknowledged(),
                alert.getAcknowledgedBy(),
                alert.getAcknowledgedAt(),
                alert.getCreatedAt()
        );
    }
}
