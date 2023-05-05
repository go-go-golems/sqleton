function addAdditionalWidgetRow() {
    // add a div.row to #additionalWidgets
    const additionalWidgetsDiv = document.querySelector('#additionalWidgets');
    const rowDiv = document.createElement('div');
    rowDiv.classList.add('row');
    additionalWidgetsDiv.appendChild(rowDiv);
    return rowDiv;
}

function addWidgets(rowDiv, widgets) {
    // add a column div
    const colDiv = document.createElement('div');
    colDiv.classList.add('column');
    rowDiv.appendChild(colDiv);

    // add the widgets to the col div
    widgets.forEach((widget) => {
        colDiv.appendChild(widget);
    })
}

function createCheckbox(labelText, id, checked) {
    const checkbox = document.createElement('input');
    checkbox.type = 'checkbox';
    checkbox.id = id;
    checkbox.checked = checked;
    const label = document.createElement('label');
    label.htmlFor = id;
    label.innerText = labelText;
    label.classList.add('label-inline');

    const rowDiv = addAdditionalWidgetRow();
    const colDiv = document.createElement('div');
    colDiv.style.height = '100%';
    rowDiv.appendChild(colDiv);
    const floatRightDiv = document.createElement('div');
    floatRightDiv.classList.add('float-right');
    rowDiv.appendChild(floatRightDiv);
    floatRightDiv.appendChild(label);
    floatRightDiv.appendChild(checkbox);
    return [ rowDiv, checkbox];
}

function setupDataTables(columnDefs, data) {
    const gridOptions = {
        columnDefs: columnDefs.map((col) => {
            return {
                headerName: col,
                field: col,
            };
        }),
        rowData: data,
        defaultColDef: {
            editable: false,
            sortable: true,
            filter: true,
            resizable: true,
        },
        sideBar: 'columns',
        onGridReady: (params) => {
            params.columnApi.autoSizeAllColumns(false);
        }
    };

    const gridDiv = document.querySelector('#tableContainer');
    new agGrid.Grid(gridDiv, gridOptions);

    const additionalWidgetsDiv = document.querySelector('#additionalWidgets');
    const [rowDiv, paginationCheckbox] = createCheckbox('Enable pagination', 'paginationCheckbox', false);
    // add a checkbox to enable pagination
    paginationCheckbox.addEventListener('change', () => {
        // enable or set pagination to 200
        gridOptions.api.setPagination(paginationCheckbox.checked)
        gridOptions.paginationPageSize = paginationCheckbox.checked ? 200 : undefined;
    });
    additionalWidgetsDiv.appendChild(rowDiv);
}