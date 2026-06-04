<template>
  <div class="app-container inspection-page">
    <div class="filter-bar">
      <el-select
        v-model="selectedBuilding"
        placeholder="选择楼栋"
        clearable
        class="building-select"
        @change="handleQuery"
      >
        <el-option v-for="b in buildings" :key="b.value" :label="b.label" :value="b.value" />
      </el-select>
      <el-button type="primary" icon="Download" @click="handleExportExcel"> 导出 Excel </el-button>
      <el-button type="danger" icon="Document" @click="handleExportPDF"> 导出 PDF </el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="roomList"
      style="width: 100%"
      :row-class-name="rowClassName"
      :default-expand-all="false"
    >
      <el-table-column type="expand">
        <template #default="{ row }">
          <div class="expand-content">
            <el-table :data="row.students" size="small" border>
              <el-table-column prop="student_name" label="姓名" align="center" min-width="100" />
              <el-table-column prop="student_id" label="学号" align="center" min-width="140" />
              <el-table-column prop="status" label="状态" align="center" min-width="100">
                <template #default="{ row: student }">
                  <el-tag :type="studentStatusType(student.status)" size="small">
                    {{ student.status }}
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="room" label="房间号" align="center" min-width="120" />
      <el-table-column prop="total_students" label="应到人数" align="center" min-width="100" />
      <el-table-column prop="present_count" label="实到人数" align="center" min-width="100" />
      <el-table-column prop="unknown_count" label="待核实人数" align="center" min-width="100" />
      <el-table-column label="状态" align="center" min-width="120">
        <template #default="{ row }">
          <el-tag :type="row.unknown_count === 0 ? 'success' : 'warning'" size="default">
            {{ row.unknown_count === 0 ? '✅ 正常' : '⚠️ 待检查' }}
          </el-tag>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { ElMessage } from 'element-plus';
import { getInspectionList } from '@/api/attendance';
import type { InspectionRoom } from '@/api/attendance';
import * as XLSX from 'xlsx';
import jsPDF from 'jspdf';
import autoTable from 'jspdf-autotable';

const buildings = [
  { label: 'A栋', value: 'A' },
  { label: 'B栋', value: 'B' },
  { label: 'C栋', value: 'C' },
  { label: 'D栋', value: 'D' },
];

const selectedBuilding = ref('');
const roomList = ref<InspectionRoom[]>([]);
const loading = ref(false);

function studentStatusType(status: string): '' | 'success' | 'warning' | 'danger' | 'info' {
  if (status === '在寝') return 'success';
  if (status === '未归') return 'danger';
  if (status === '晚归') return 'warning';
  return 'info';
}

function rowClassName({ row }: { row: InspectionRoom }): string {
  return row.unknown_count > 0 ? 'warning-row' : '';
}

async function handleQuery() {
  loading.value = true;
  try {
    const res: any = await getInspectionList(selectedBuilding.value || undefined);
    const data = res.data ?? res;
    roomList.value = Array.isArray(data) ? data : [];
  } catch {
    ElMessage.error('获取查寝名单失败');
    roomList.value = [];
  } finally {
    loading.value = false;
  }
}

function handleExportExcel() {
  if (!roomList.value.length) {
    ElMessage.warning('暂无数据可导出');
    return;
  }

  const rows: Record<string, string>[] = [];
  roomList.value.forEach(room => {
    room.students.forEach(student => {
      rows.push({
        楼栋: room.building,
        房间号: room.room,
        学生姓名: student.student_name,
        学号: student.student_id,
        状态: student.status,
      });
    });
  });

  const ws = XLSX.utils.json_to_sheet(rows);
  const colWidths = [{ wch: 8 }, { wch: 12 }, { wch: 12 }, { wch: 18 }, { wch: 8 }];
  ws['!cols'] = colWidths;

  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, '查寝名单');
  XLSX.writeFile(wb, '查寝名单.xlsx');
}

function handleExportPDF() {
  if (!roomList.value.length) {
    ElMessage.warning('暂无数据可导出');
    return;
  }

  const doc = new jsPDF({ orientation: 'landscape' });

  const head = [['房间号', '学生姓名', '学号', '状态']];
  const body: string[][] = [];
  roomList.value.forEach(room => {
    room.students.forEach(student => {
      body.push([room.room, student.student_name, student.student_id, student.status]);
    });
  });

  autoTable(doc, {
    head,
    body,
    startY: 20,
    styles: { fontSize: 10 },
    headStyles: { fillColor: [24, 144, 255], textColor: 255 },
    alternateRowStyles: { fillColor: [245, 247, 250] },
    didDrawPage: data => {
      if (data.pageNumber === 1) {
        doc.setFontSize(16);
        doc.text('查寝名单', doc.internal.pageSize.getWidth() / 2, 14, { align: 'center' });
      }
    },
  });

  doc.save('查寝名单.pdf');
}

onMounted(() => {
  handleQuery();
});
</script>

<style scoped lang="scss">
.inspection-page {
  .filter-bar {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
  }

  .building-select {
    width: 160px;
  }

  :deep(.warning-row) {
    background-color: #fff7e6 !important;
  }

  .expand-content {
    padding: 8px 16px 16px 48px;
  }
}
</style>
