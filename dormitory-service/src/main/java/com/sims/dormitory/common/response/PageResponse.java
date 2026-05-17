package com.sims.dormitory.common.response;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;
import lombok.ToString;

import java.util.List;

@Getter
@Setter
@ToString
@NoArgsConstructor
@AllArgsConstructor
public class PageResponse<T> {

    private List<T> items;
    private long total;
    private int page;
    private int size;
    private int pages;

    public static <T> PageResponse<T> of(List<T> items, long total, int page, int size) {
        int pages = (size > 0) ? (int) Math.ceil((double) total / size) : 0;
        return new PageResponse<>(items, total, page, size, pages);
    }
}
