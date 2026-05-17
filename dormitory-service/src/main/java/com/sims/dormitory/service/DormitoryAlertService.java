package com.sims.dormitory.service;

import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.model.dto.AlertDTO;
import com.sims.dormitory.model.entity.DormAlert;

import java.util.List;

public interface DormitoryAlertService {

    DormAlert createAlert(DormAlert alert);

    void acknowledgeAlert(Long id, String acknowledgedBy);

    Page<AlertDTO> getAlerts(Long buildingId, String alertType, Boolean acknowledged,
                             int page, int size);

    long getAlertCount(Long buildingId, Boolean acknowledged);
}
