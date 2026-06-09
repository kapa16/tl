#Область СлужебныеПроцедурыИФункции

Функция ExecuteCommand(loginData, commandName, requestData, responseData, resultCode, resultDescription)
	
	Попытка
		Возврат уатМобильноеПриложениеВодителяСервер.ОбработатьМетодExecuteCommand(loginData, commandName, requestData, responseData, resultCode, resultDescription);
	Исключение
		уатМобильноеПриложениеВодителяСервер.ЗаписатьСообщениеОбОшибке("ExecuteCommand", ОписаниеОшибки());
		responseData      = "<response/>"; 
		resultDescription = ОписаниеОшибки();
		resultCode        = 29999;
		
		Возврат Ложь;
	КонецПопытки;
КонецФункции

#КонецОбласти
