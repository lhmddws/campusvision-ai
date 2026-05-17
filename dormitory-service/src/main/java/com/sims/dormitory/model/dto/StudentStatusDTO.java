package com.sims.dormitory.model.dto;

import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

@Getter
@Setter
@ToString
public class StudentStatusDTO {
    private Long buildingId;
    private Long roomId;
    private String status;
}
